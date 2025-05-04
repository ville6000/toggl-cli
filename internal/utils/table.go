package utils

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

// RenderTable renders a table to the standard output.
// It takes headers and rows as parameters.
func RenderTable(
	title string,
	headers []interface{},
	rows [][]interface{},
	footer table.Row,
) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

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
