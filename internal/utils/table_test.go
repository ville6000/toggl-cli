package utils

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Errorf("close pipe reader: %v", err)
		}
	})
	old := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}

func TestRenderTable_ContainsHeaders(t *testing.T) {
	out := captureStdout(t, func() {
		RenderTable(
			"",
			[]interface{}{"ID", "NAME"},
			[][]interface{}{{1, "Alpha"}},
			nil,
		)
	})
	if !strings.Contains(out, "ID") || !strings.Contains(out, "NAME") {
		t.Errorf("output missing headers: %q", out)
	}
}

func TestRenderTable_ContainsRows(t *testing.T) {
	out := captureStdout(t, func() {
		RenderTable(
			"",
			[]interface{}{"ID", "NAME"},
			[][]interface{}{
				{1, "Alpha"},
				{2, "Beta"},
			},
			nil,
		)
	})
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "Beta") {
		t.Errorf("output missing row data: %q", out)
	}
}

func TestRenderTable_WithTitle(t *testing.T) {
	withTitle := captureStdout(t, func() {
		RenderTable("Title", []interface{}{"COL"}, [][]interface{}{{"val"}}, nil)
	})
	withoutTitle := captureStdout(t, func() {
		RenderTable("", []interface{}{"COL"}, [][]interface{}{{"val"}}, nil)
	})
	// Output with a title should be longer because of the extra title rows.
	if len(withTitle) <= len(withoutTitle) {
		t.Errorf("expected title to add output lines (with=%d, without=%d)", len(withTitle), len(withoutTitle))
	}
}

func TestRenderTable_NoTitle(t *testing.T) {
	// Should not panic and should still render headers/rows.
	out := captureStdout(t, func() {
		RenderTable(
			"",
			[]interface{}{"COL"},
			[][]interface{}{{"val"}},
			nil,
		)
	})
	if !strings.Contains(out, "COL") {
		t.Errorf("output missing header: %q", out)
	}
}

func TestRenderTable_WithFooter(t *testing.T) {
	out := captureStdout(t, func() {
		RenderTable(
			"",
			[]interface{}{"NUM"},
			[][]interface{}{{1}, {2}},
			table.Row{"Total: 2"},
		)
	})
	// The table library uppercases footer text.
	if !strings.Contains(strings.ToUpper(out), "TOTAL: 2") {
		t.Errorf("output missing footer: %q", out)
	}
}

func TestRenderTable_EmptyRows(t *testing.T) {
	// Should not panic with no rows.
	captureStdout(t, func() {
		RenderTable(
			"Empty",
			[]interface{}{"COL"},
			[][]interface{}{},
			nil,
		)
	})
}
