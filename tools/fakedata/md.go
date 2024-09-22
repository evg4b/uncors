package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/samber/lo"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/evg4b/uncors/pkg/fakedata"
)

type MdTableRow struct {
	Type        []string
	Description []string
	Params      []string
	Example     []string
}

func generateMdData() {
	rows := make([]MdTableRow, 0)
	for _, typeKey := range fakedata.GetTypes() {
		info := gofakeit.GetFuncLookup(typeKey)
		if info == nil {
			continue
		}
		rows = append(rows, MdTableRow{
			Type:        []string{typeKey},
			Description: []string{info.Description},
			Example:     []string{info.Example},
			Params: lo.Map(info.Params, func(param gofakeit.Param, _ int) string {
				return fmt.Sprintf("%s (%s) - %s", param.Field, param.Type, param.Description)
			}),
		})
	}

	mdFile, err := os.OpenFile("tools/fakedata/docs.md", os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}

	if _, err = mdFile.WriteString("| Type | Description | Params | Example |\n"); err != nil {
		panic(err)
	}

	if _, err = mdFile.WriteString("| ---- | ----------- | ------- | ------- |\n"); err != nil {
		panic(err)
	}

	for _, row := range rows {
		if _, err = fmt.Fprintf(
			mdFile,
			"| %s | %s | %s | %s |\n",
			f(row.Type),
			f(row.Description),
			f(row.Params),
			f(row.Example),
		); err != nil {
			panic(err)
		}
	}
}

func f(lines []string) string {
	return strings.ReplaceAll(strings.Join(lines, "\n"), "\n", "<br>")
}
