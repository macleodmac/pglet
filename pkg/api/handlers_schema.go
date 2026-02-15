package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/macleodmac/pglet/pkg/client"
)

func (s *Server) ListSchemas(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	schemas, err := cl.Schemas()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, schemas)
}

func (s *Server) ListObjects(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	objects, err := cl.Objects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
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
	c.JSON(http.StatusOK, result)
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

func (s *Server) GetTableColumns(c *gin.Context, table string) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	cols, err := cl.TableColumns(table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
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
	c.JSON(http.StatusOK, result)
}

func (s *Server) GetTableRows(c *gin.Context, table string, params GetTableRowsParams) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}

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

	result, total, err := cl.TableRows(c.Request.Context(), table, limit, offset, sortCol, sortOrd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, TableRowsResult{
		Columns: result.Columns, ColumnTypes: result.ColumnTypes,
		Rows: toNullableRows(result.Rows), TotalCount: total,
		Page: offset / limit, PageSize: limit,
	})
}

func (s *Server) GetTableInfo(c *gin.Context, table string) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	info, err := cl.TableInfo(table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, TableInfo{
		TotalSize: info.TotalSize, TableSize: info.TableSize,
		IndexSize: info.IndexSize, RowEstimate: info.RowEstimate,
	})
}

func (s *Server) GetTableIndexes(c *gin.Context, table string) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	indexes, err := cl.TableIndexes(table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	result := make([]TableIndex, len(indexes))
	for i, idx := range indexes {
		result[i] = TableIndex{
			Name: idx.Name, Definition: idx.Definition,
			IsUnique: idx.IsUnique, IsPrimary: idx.IsPrimary,
		}
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) GetTableConstraints(c *gin.Context, table string) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	constraints, err := cl.TableConstraints(table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	result := make([]TableConstraint, len(constraints))
	for i, con := range constraints {
		result[i] = TableConstraint{
			Name: con.Name, Type: con.Type, Definition: con.Definition,
		}
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) GetFunctionDefinition(c *gin.Context, function string) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	schema, name := splitQualifiedName(function)
	fd, err := cl.FunctionDefinition(schema, name)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "function not found"})
		return
	}
	c.JSON(http.StatusOK, FunctionDefinition{
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

func (s *Server) GetTablesStats(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	result, err := cl.TablesStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, toQueryResult(result))
}

func (s *Server) GetActivity(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	activities, err := cl.Activity()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
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
	c.JSON(http.StatusOK, result)
}

func (s *Server) GetServerSettings(c *gin.Context) {
	cl := s.getClient()
	if cl == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "not connected"})
		return
	}
	result, err := cl.ServerSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, toQueryResult(result))
}

func toQueryResult(r *client.QueryResult) QueryResult {
	return QueryResult{
		Columns: r.Columns, ColumnTypes: r.ColumnTypes,
		Rows: toNullableRows(r.Rows), RowCount: r.RowCount,
		DurationMs: r.DurationMs,
	}
}

func toNullableRows(rows [][]any) [][]CellValue {
	result := make([][]CellValue, len(rows))
	for i, row := range rows {
		result[i] = make([]CellValue, len(row))
		for j, v := range row {
			if v == nil {
				result[i][j] = nil
			} else {
				s := fmt.Sprintf("%v", v)
				result[i][j] = &s
			}
		}
	}
	return result
}
