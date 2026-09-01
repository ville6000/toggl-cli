package utils

import (
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
)

// RenderTable renders a table to out. The title and footer are optional:
// an empty title or a nil footer omits that section.
func RenderTable(
	out io.Writer,
	title string,
	headers []interface{},
	rows [][]interface{},
	footer table.Row,
) {
	t := table.NewWriter()
	t.SetOutputMirror(out)

	if title != "" {
		t.SetTitle(title)
	}

	t.AppendHeader(headers)

	for _, row := range rows {
		t.AppendRow(row)
	}

	if footer != nil {
		t.AppendFooter(footer)
	}

	t.Render()
}
