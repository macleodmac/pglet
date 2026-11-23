package client

import (
	"database/sql/driver"
	"encoding/hex"
	"math"
	"strconv"
	"strings"
	"time"
)

// formatRow renders a row of driver values as PostgreSQL's text output would,
// so values round-trip instead of being re-encoded by Go's formatting.
func formatRow(vals []driver.Value, typeNames []string) []*string {
	row := make([]*string, len(vals))
	for i, v := range vals {
		if v == nil {
			continue
		}
		s := formatValue(v, typeNames[i])
		row[i] = &s
	}
	return row
}

func formatValue(v driver.Value, typeName string) string {
	switch v := v.(type) {
	case []byte:
		if typeName == "BYTEA" {
			return `\x` + hex.EncodeToString(v)
		}
		return string(v)
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return formatFloat(v)
	case time.Time:
		return formatTime(v, typeName)
	}
	return ""
}

func formatFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	if abs := math.Abs(f); abs == 0 || (abs >= 1e-4 && abs < 1e15) {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func formatTime(t time.Time, typeName string) string {
	switch typeName {
	case "DATE":
		return t.Format("2006-01-02")
	case "TIMESTAMP":
		return t.Format("2006-01-02 15:04:05.999999")
	case "TIME":
		return t.Format("15:04:05.999999")
	case "TIMETZ":
		return t.Format("15:04:05.999999") + formatOffset(t)
	default:
		return t.Format("2006-01-02 15:04:05.999999") + formatOffset(t)
	}
}

// formatOffset renders a UTC offset the way PostgreSQL does: +00, +05:30.
func formatOffset(t time.Time) string {
	return strings.TrimSuffix(t.Format("-07:00"), ":00")
}

// JSONKind says how a column's text values should be written to JSON: as raw
// numbers or booleans when that is lossless in JavaScript, otherwise strings.
func JSONKind(typeName string) string {
	switch typeName {
	case "INT2", "INT4", "FLOAT4", "FLOAT8":
		return "number"
	case "BOOL":
		return "bool"
	case "JSON", "JSONB":
		return "json"
	}
	return "string"
}
