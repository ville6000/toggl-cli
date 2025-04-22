package utils

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"os"
)

func RenderTable(headers []interface{}, rows [][]interface{}) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(headers)

	for _, row := range rows {
		t.AppendRow(row)
	}

	t.Render()
}
