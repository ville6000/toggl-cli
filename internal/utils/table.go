package utils

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"os"
)
// RenderTable renders a table to the standard output.
// It takes headers and rows as parameters.
func RenderTable(headers []interface{}, rows [][]interface{}) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(headers)

	for _, row := range rows {
		t.AppendRow(row)
	}

	t.Render()
}
