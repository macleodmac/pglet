package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/macleodmac/pglet/internal/ai"
	"github.com/macleodmac/pglet/internal/client"
	"github.com/macleodmac/pglet/internal/repository"
	"github.com/macleodmac/pglet/internal/service"
)

func (s *Server) ListSchemas(w http.ResponseWriter, r *http.Request) {
	schemas, err := s.svc.Schemas(r.Context())
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, schemas)
}

func (s *Server) ListObjects(w http.ResponseWriter, r *http.Request) {
	objects, err := s.svc.Objects(r.Context())
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}

	result := make(map[string]SchemaGroup)
	for schema, group := range objects {
		result[schema] = SchemaGroup{
			Tables:            toSchemaObjects(group.Tables),
			Views:             toSchemaObjects(group.Views),
			MaterializedViews: toSchemaObjects(group.MaterializedViews),
			Functions:         toSchemaObjects(group.Functions),
			Sequences:         toSchemaObjects(group.Sequences),
			Types:             toSchemaObjects(group.Types),
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func toSchemaObjects(objs []client.SchemaObject) []SchemaObject {
	result := make([]SchemaObject, len(objs))
	for i, o := range objs {
		result[i] = SchemaObject{Name: o.Name, Schema: o.Schema, Type: o.Type}
		if o.Comment != "" {
			result[i].Comment = &o.Comment
		}
	}
	return result
}

func (s *Server) GetTableColumns(w http.ResponseWriter, r *http.Request, table string) {
	cols, err := s.svc.TableColumns(r.Context(), table)
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}

	result := make([]Column, len(cols))
	for i, col := range cols {
		result[i] = Column{
			Name: col.Name, Type: col.Type, Nullable: col.Nullable,
			DefaultValue: col.DefaultValue, Position: col.Position,
			IsPrimaryKey: col.IsPrimaryKey,
		}
		if col.Comment != "" {
			result[i].Comment = &col.Comment
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) GetTableRows(w http.ResponseWriter, r *http.Request, table string, params GetTableRowsParams) {
	limit, offset := 100, 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	if params.Offset != nil {
		offset = *params.Offset
	}
	sortCol, sortOrd := "", ""
	if params.SortColumn != nil {
		sortCol = *params.SortColumn
	}
	if params.SortOrder != nil {
		sortOrd = string(*params.SortOrder)
	}

	result, total, err := s.svc.TableRows(r.Context(), table, limit, offset, sortCol, sortOrd)
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}

	writeJSON(w, http.StatusOK, TableRowsResult{
		Columns: result.Columns, ColumnTypes: result.ColumnTypes,
		Rows: nonNilRows(result.Rows), TotalCount: total,
		Page: offset / limit, PageSize: limit,
	})
}

func (s *Server) GetTableInfo(w http.ResponseWriter, r *http.Request, table string) {
	info, err := s.svc.TableInfo(r.Context(), table)
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, TableInfo{
		TotalSize: info.TotalSize, TableSize: info.TableSize,
		IndexSize: info.IndexSize, RowEstimate: info.RowEstimate,
	})
}

func (s *Server) GetTableIndexes(w http.ResponseWriter, r *http.Request, table string) {
	indexes, err := s.svc.TableIndexes(r.Context(), table)
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}
	result := make([]TableIndex, len(indexes))
	for i, idx := range indexes {
		result[i] = TableIndex{
			Name: idx.Name, Definition: idx.Definition,
			IsUnique: idx.IsUnique, IsPrimary: idx.IsPrimary,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) GetTableConstraints(w http.ResponseWriter, r *http.Request, table string) {
	constraints, err := s.svc.TableConstraints(r.Context(), table)
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}
	result := make([]TableConstraint, len(constraints))
	for i, con := range constraints {
		result[i] = TableConstraint{
			Name: con.Name, Type: con.Type, Definition: con.Definition,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) GetFunctionDefinition(w http.ResponseWriter, r *http.Request, function string) {
	schema, name := splitQualifiedName(function)
	fd, err := s.svc.FunctionDefinition(r.Context(), schema, name)
	if err != nil {
		if errors.Is(err, service.ErrNotConnected) {
			writeErr(w, http.StatusBadRequest, err)
		} else {
			writeErrMsg(w, http.StatusNotFound, "function not found")
		}
		return
	}
	writeJSON(w, http.StatusOK, FunctionDefinition{
		Name: fd.Name, Schema: fd.Schema, Definition: fd.Definition,
		Language: fd.Language, Arguments: fd.Arguments, ReturnType: fd.ReturnType,
		Volatility: fd.Volatility, Kind: fd.Kind,
	})
}

func splitQualifiedName(name string) (string, string) {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", parts[0]
}

func (s *Server) GetTablesStats(w http.ResponseWriter, r *http.Request) {
	result, err := s.svc.TablesStats(r.Context())
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, toQueryResult(result))
}

func (s *Server) GetActivity(w http.ResponseWriter, r *http.Request) {
	activities, err := s.svc.Activity(r.Context())
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}
	result := make([]Activity, len(activities))
	for i, a := range activities {
		result[i] = Activity{
			Pid: a.PID, Database: a.Database, User: a.User,
			Application: a.Application, ClientAddr: a.ClientAddr,
			State: a.State, Query: a.Query, Duration: a.Duration,
			WaitEvent: a.WaitEvent, WaitEventType: a.WaitEventType,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) GetServerSettings(w http.ResponseWriter, r *http.Request) {
	result, err := s.svc.ServerSettings(r.Context())
	if err != nil {
		writeErr(w, svcStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, toQueryResult(result))
}

func toQueryResult(qr *client.QueryResult) QueryResult {
	res := QueryResult{
		Columns: qr.Columns, ColumnTypes: qr.ColumnTypes,
		Rows: nonNilRows(qr.Rows), RowCount: qr.RowCount,
		DurationMs: qr.DurationMs, RowsAffected: qr.RowsAffected,
	}
	if qr.CommandTag != "" {
		res.CommandTag = &qr.CommandTag
	}
	if qr.Truncated {
		res.Truncated = &qr.Truncated
	}
	return res
}

// nonNilRows keeps an empty result encoding as [] rather than null.
func nonNilRows(rows [][]*string) [][]*CellValue {
	if rows == nil {
		return [][]*CellValue{}
	}
	return rows
}

// svcStatus maps service errors to HTTP status codes.
func svcStatus(err error) int {
	if errors.Is(err, service.ErrNotConnected) || errors.Is(err, service.ErrNoTransaction) ||
		errors.Is(err, client.ErrTxAborted) {
		return http.StatusBadRequest
	}
	if errors.Is(err, service.ErrTxConflict) || errors.Is(err, service.ErrQueryRunning) {
		return http.StatusConflict
	}
	if errors.Is(err, client.ErrPoolExhausted) {
		return http.StatusServiceUnavailable
	}
	if errors.Is(err, repository.ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ai.ErrRefused) {
		return http.StatusUnprocessableEntity
	}
	return http.StatusInternalServerError
}
