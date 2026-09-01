package utils

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
)

// render returns what RenderTable writes for the given arguments.
func render(title string, headers []interface{}, rows [][]interface{}, footer table.Row) string {
	var buf bytes.Buffer
	RenderTable(&buf, title, headers, rows, footer)
	return buf.String()
}

func TestRenderTable_ContainsHeaders(t *testing.T) {
	out := render("", []interface{}{"ID", "NAME"}, [][]interface{}{{1, "Alpha"}}, nil)
	if !strings.Contains(out, "ID") || !strings.Contains(out, "NAME") {
		t.Errorf("output missing headers: %q", out)
	}
}

func TestRenderTable_ContainsRows(t *testing.T) {
	out := render(
		"",
		[]interface{}{"ID", "NAME"},
		[][]interface{}{
			{1, "Alpha"},
			{2, "Beta"},
		},
		nil,
	)
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "Beta") {
		t.Errorf("output missing row data: %q", out)
	}
}

func TestRenderTable_WithTitle(t *testing.T) {
	withTitle := render("Title", []interface{}{"COL"}, [][]interface{}{{"val"}}, nil)
	withoutTitle := render("", []interface{}{"COL"}, [][]interface{}{{"val"}}, nil)
	// Output with a title should be longer because of the extra title rows.
	if len(withTitle) <= len(withoutTitle) {
		t.Errorf("expected title to add output lines (with=%d, without=%d)", len(withTitle), len(withoutTitle))
	}
}

func TestRenderTable_NoTitle(t *testing.T) {
	// Should not panic and should still render headers/rows.
	out := render("", []interface{}{"COL"}, [][]interface{}{{"val"}}, nil)
	if !strings.Contains(out, "COL") {
		t.Errorf("output missing header: %q", out)
	}
}

func TestRenderTable_WithFooter(t *testing.T) {
	out := render("", []interface{}{"NUM"}, [][]interface{}{{1}, {2}}, table.Row{"Total: 2"})
	// The table library uppercases footer text.
	if !strings.Contains(strings.ToUpper(out), "TOTAL: 2") {
		t.Errorf("output missing footer: %q", out)
	}
}

func TestRenderTable_EmptyRows(t *testing.T) {
	// Should not panic with no rows.
	render("Empty", []interface{}{"COL"}, [][]interface{}{}, nil)
}

func TestRenderTable_WritesToGivenWriter(t *testing.T) {
	var buf bytes.Buffer
	RenderTable(&buf, "Title", []interface{}{"COL"}, [][]interface{}{{"val"}}, nil)
	if buf.Len() == 0 {
		t.Error("RenderTable wrote nothing to the provided writer")
	}
}
