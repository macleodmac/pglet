package api

import (
	"crypto/subtle"
	"net"
	"net/http"
	"slices"
	"strings"
)

// CSRFHeader must be present on every API request. Browsers only let a page
// send a custom header cross-origin after a CORS preflight, which pglet
// refuses for origins not on the --cors-origin list.
const CSRFHeader = "X-Pglet"

// Guard protects the API from other websites and other users.
type Guard struct {
	// Token, when set, must arrive in the session cookie. Visiting any page
	// with ?token=<Token> sets that cookie.
	Token string
	// LoopbackOnly rejects requests whose Host is not a loopback name, which
	// defeats DNS rebinding.
	LoopbackOnly bool
	// Prefix is the URL prefix pglet is served under, "" for the root.
	Prefix     string
	CookieName string
}

func (g Guard) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.LoopbackOnly && !isLoopbackHost(r.Host) {
			writeErrMsg(w, http.StatusForbidden, "pglet only answers requests addressed to localhost")
			return
		}

		if !strings.HasPrefix(r.URL.Path, g.Prefix+"/api/") {
			if g.Token != "" && g.validToken(r.URL.Query().Get("token")) {
				g.setCookie(w)
				u := *r.URL
				q := u.Query()
				q.Del("token")
				u.RawQuery = q.Encode()
				http.Redirect(w, r, u.String(), http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if r.Header.Get(CSRFHeader) == "" {
			writeErrMsg(w, http.StatusForbidden, "missing "+CSRFHeader+" header")
			return
		}
		if g.Token != "" {
			c, err := r.Cookie(g.CookieName)
			if err != nil || !g.validToken(c.Value) {
				writeErrMsg(w, http.StatusUnauthorized, "not signed in: open pglet with the link printed in its terminal")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (g Guard) validToken(t string) bool {
	return t != "" && subtle.ConstantTimeCompare([]byte(t), []byte(g.Token)) == 1
}

func (g Guard) setCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     g.CookieName,
		Value:    g.Token,
		Path:     g.Prefix + "/",
		MaxAge:   400 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func isLoopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.Trim(host, "[]"), ".")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// CorsMiddleware allows the listed origins to call the API with credentials.
// Any other origin gets no CORS headers, so browsers block it.
func CorsMiddleware(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && slices.Contains(origins, origin) {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type, "+CSRFHeader)
				h.Add("Vary", "Origin")
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
