package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) ExportQuery(w http.ResponseWriter, r *http.Request) {
	var req ExportRequest
	if err := readJSON(r, &req); err != nil {
		writeErrMsg(w, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := s.svc.ExportQuery(r.Context(), req.Query)
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}

	switch req.Format {
	case Csv:
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=export.csv")
		cw := csv.NewWriter(w)

		if err := cw.Write(result.Columns); err != nil {
			return
		}

		for _, row := range result.Rows {
			record := make([]string, len(row))
			for i, v := range row {
				if v == nil {
					record[i] = ""
				} else {
					record[i] = fmt.Sprintf("%v", v)
				}
			}
			if err := cw.Write(record); err != nil {
				return
			}
		}
		cw.Flush()

	case Json:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=export.json")

		records := make([]map[string]interface{}, len(result.Rows))
		for i, row := range result.Rows {
			record := make(map[string]interface{})
			for j, v := range row {
				if j < len(result.Columns) {
					record[result.Columns[j]] = v
				}
			}
			records[i] = record
		}

		json.NewEncoder(w).Encode(records)

	default:
		writeErrMsg(w, http.StatusBadRequest, "unsupported format, use csv or json")
	}
}
