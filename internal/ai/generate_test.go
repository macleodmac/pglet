package ai

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatSchema(t *testing.T) {
	got := formatSchema([]Table{{
		Name:    "public.orders",
		Kind:    "table",
		Comment: "Customer orders,\n  one per checkout",
		Columns: []Column{
			{Name: "id", Type: "bigint", PrimaryKey: true},
			{Name: "customer_id", Type: "integer", References: "public.customers.id"},
			{Name: "state", Type: "order_status", EnumValues: []string{"pending", "it's paid"}, Comment: "Lifecycle"},
		},
	}})

	want := `public.orders (table) -- Customer orders, one per checkout
  id int8 PK
  customer_id int -> public.customers.id
  state order_status enum('pending', 'it''s paid') -- Lifecycle
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatSchemaTruncates(t *testing.T) {
	values := make([]string, maxEnumValues+5)
	for i := range values {
		values[i] = fmt.Sprint(i)
	}
	got := formatSchema([]Table{{
		Name:    "public.t",
		Kind:    "table",
		Comment: strings.Repeat("x", maxCommentLen+10),
		Columns: []Column{{Name: "c", Type: "e", EnumValues: values}},
	}})

	if !strings.Contains(got, "'29', …)") || strings.Contains(got, "'30'") {
		t.Errorf("enum values not capped at %d: %s", maxEnumValues, got)
	}
	if !strings.Contains(got, strings.Repeat("x", maxCommentLen)+"…\n") {
		t.Errorf("comment not truncated: %s", got)
	}
}
