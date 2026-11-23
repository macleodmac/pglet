package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGuard(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })
	g := Guard{Token: "secret", LoopbackOnly: true, Prefix: "/pg", CookieName: "pglet_token_8081"}
	h := g.Middleware(ok)
	cookie := &http.Cookie{Name: "pglet_token_8081", Value: "secret"}

	serve := func(method, target, host string, mod func(*http.Request)) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, target, nil)
		r.Host = host
		if mod != nil {
			mod(r)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	signedIn := func(r *http.Request) {
		r.AddCookie(cookie)
		r.Header.Set(CSRFHeader, "1")
	}

	cases := []struct {
		name   string
		method string
		target string
		host   string
		mod    func(*http.Request)
		want   int
	}{
		{"signed-in API call", "POST", "/pg/api/query", "localhost:8081", signedIn, http.StatusTeapot},
		{"IPv6 loopback", "POST", "/pg/api/query", "[::1]:8081", signedIn, http.StatusTeapot},
		{"rebound hostname", "POST", "/pg/api/query", "rebind.attacker.example:8081", signedIn, http.StatusForbidden},
		{"no CSRF header", "POST", "/pg/api/disconnect", "localhost:8081", func(r *http.Request) { r.AddCookie(cookie) }, http.StatusForbidden},
		{"no cookie", "GET", "/pg/api/ai/suggestions", "localhost:8081", func(r *http.Request) { r.Header.Set(CSRFHeader, "1") }, http.StatusUnauthorized},
		{"wrong cookie", "GET", "/pg/api/info", "localhost:8081", func(r *http.Request) {
			r.Header.Set(CSRFHeader, "1")
			r.AddCookie(&http.Cookie{Name: "pglet_token_8081", Value: "nope"})
		}, http.StatusUnauthorized},
		{"static page needs no token", "GET", "/pg/", "localhost:8081", nil, http.StatusTeapot},
		{"token link sets cookie", "GET", "/pg/?token=secret", "localhost:8081", nil, http.StatusSeeOther},
		{"wrong token link", "GET", "/pg/?token=nope", "localhost:8081", nil, http.StatusTeapot},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := serve(tc.method, tc.target, tc.host, tc.mod)
			if w.Code != tc.want {
				t.Fatalf("status %d, want %d: %s", w.Code, tc.want, w.Body)
			}
			if tc.want == http.StatusSeeOther {
				if loc := w.Header().Get("Location"); loc != "/pg/" {
					t.Errorf("redirect to %q", loc)
				}
				c := w.Result().Cookies()
				if len(c) != 1 || c[0].Value != "secret" || !c[0].HttpOnly || c[0].SameSite != http.SameSiteStrictMode || c[0].Path != "/pg/" {
					t.Errorf("cookie = %+v", c)
				}
			}
		})
	}
}

func TestCorsMiddleware(t *testing.T) {
	h := CorsMiddleware([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	for origin, allowed := range map[string]bool{"http://localhost:5173": true, "https://evil.example": false} {
		r := httptest.NewRequest("OPTIONS", "/api/query", nil)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if got := w.Header().Get("Access-Control-Allow-Origin") == origin; got != allowed {
			t.Errorf("%s: allowed = %v", origin, got)
		}
	}
}

func TestJSONValue(t *testing.T) {
	s := func(v string) *string { return &v }
	cases := []struct {
		v    *string
		kind string
		want string
	}{
		{nil, "number", "null"},
		{s("42"), "number", "42"},
		{s("NaN"), "number", `"NaN"`},
		{s("9007199254740993"), "string", `"9007199254740993"`},
		{s("true"), "bool", "true"},
		{s(`{"a":9007199254740993}`), "json", `{"a":9007199254740993}`},
		{s(`say "hi"`), "string", `"say \"hi\""`},
	}
	for _, tc := range cases {
		if got := string(jsonValue(tc.v, tc.kind)); got != tc.want {
			t.Errorf("jsonValue(%v, %s) = %s, want %s", tc.v, tc.kind, got, tc.want)
		}
	}
}
