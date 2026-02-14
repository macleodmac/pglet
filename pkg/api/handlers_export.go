package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) ExportQuery(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}

	var req ExportRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	result, err := cl.QueryWithContext(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	switch req.Format {
	case Csv:
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=export.csv")
		w := csv.NewWriter(c.Writer)

		// Write header row
		if err := w.Write(result.Columns); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to write CSV header"})
			return
		}

		// Write data rows
		for _, row := range result.Rows {
			record := make([]string, len(row))
			for i, v := range row {
				if v == nil {
					record[i] = ""
				} else {
					record[i] = fmt.Sprintf("%v", v)
				}
			}
			if err := w.Write(record); err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to write CSV row"})
				return
			}
		}
		w.Flush()

		if err := w.Error(); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "CSV write error"})
			return
		}

	case Json:
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=export.json")

		// Build array of objects
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

		if err := json.NewEncoder(c.Writer).Encode(records); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to encode JSON"})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "unsupported format, use csv or json"})
	}
}
