package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/macleodmac/pglet/internal/client"
)

// ExportQuery streams rows as they arrive. Once the first row is written the
// status is committed, so a later failure can only truncate the download.
func (s *Server) ExportQuery(w http.ResponseWriter, r *http.Request) {
	var req ExportRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	var onHeader func(columns, types []string) error
	var onRow func([]*string) error
	var finish func()

	switch req.Format {
	case Csv:
		cw := csv.NewWriter(w)
		onHeader = func(columns, _ []string) error {
			w.Header().Set("Content-Type", "text/csv")
			w.Header().Set("Content-Disposition", "attachment; filename=export.csv")
			return cw.Write(columns)
		}
		onRow = func(row []*string) error {
			record := make([]string, len(row))
			for i, v := range row {
				if v != nil {
					record[i] = *v
				}
			}
			return cw.Write(record)
		}
		finish = cw.Flush

	case Json:
		var columns, kinds []string
		first, started := true, false
		onHeader = func(cols, types []string) error {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", "attachment; filename=export.json")
			columns, started = cols, true
			kinds = make([]string, len(types))
			for i, t := range types {
				kinds[i] = client.JSONKind(t)
			}
			_, err := w.Write([]byte("["))
			return err
		}
		onRow = func(row []*string) error {
			var buf bytes.Buffer
			if !first {
				buf.WriteByte(',')
			}
			first = false
			buf.WriteByte('{')
			for i, v := range row {
				if i > 0 {
					buf.WriteByte(',')
				}
				key, _ := json.Marshal(columns[i])
				buf.Write(key)
				buf.WriteByte(':')
				buf.Write(jsonValue(v, kinds[i]))
			}
			buf.WriteByte('}')
			_, err := w.Write(buf.Bytes())
			return err
		}
		finish = func() {
			if !started {
				return
			}
			w.Write([]byte("]\n"))
		}

	default:
		writeErrMsg(w, http.StatusBadRequest, "unsupported format, use csv or json")
		return
	}

	if err := s.svc.ExportQuery(r.Context(), req.Query, onHeader, onRow); err != nil {
		if w.Header().Get("Content-Disposition") == "" {
			writeErr(w, exportStatus(err), err)
		}
		return
	}
	finish()
}

// jsonValue writes numbers and booleans raw only when JavaScript can hold
// them exactly; everything else, including int8 and numeric, stays a string.
func jsonValue(v *string, kind string) []byte {
	if v == nil {
		return []byte("null")
	}
	switch kind {
	case "number":
		if _, err := strconv.ParseFloat(*v, 64); err == nil && json.Valid([]byte(*v)) {
			return []byte(*v)
		}
	case "bool":
		return []byte(*v)
	case "json":
		if json.Valid([]byte(*v)) {
			return []byte(*v)
		}
	}
	b, _ := json.Marshal(*v)
	return b
}

func exportStatus(err error) int {
	if status := svcStatus(err); status != http.StatusInternalServerError {
		return status
	}
	return http.StatusBadRequest
}
