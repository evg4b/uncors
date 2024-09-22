package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"

	"github.com/samber/lo"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/evg4b/uncors/pkg/fakedata"
)

type MdTableRow struct {
	Type        string
	Description string
	Params      []string
	Example     string
	Group       string
}

var groups = map[string][]string{
	"Numbers": {
		"number",
		"int",
		"intn",
		"int8",
		"int16",
		"int32",
		"int64",
		"uint",
		"uintn",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"float32",
		"float32range",
		"float64",
		"float64range",
	},
}

func generateMdData() {
	rows := make([]MdTableRow, 0)
	groupsMap := make(map[string]string)
	lo.ForEach(lo.Keys(groups), func(group string, _ int) {
		lo.ForEach(groups[group], func(item string, _ int) {
			groupsMap[item] = group
		})
	})

	for _, typeKey := range fakedata.GetTypes() {
		info := gofakeit.GetFuncLookup(typeKey)
		if info == nil {
			log.Warnf("Type %s not found in fakedata", typeKey)

			continue
		}

		rows = append(rows, MdTableRow{
			Type:        typeKey,
			Description: info.Description,
			Example:     info.Example,
			Group:       groupsMap[typeKey],
			Params: lo.Map(info.Params, func(param gofakeit.Param, _ int) string {
				return fmt.Sprintf("%s (%s) - %s", param.Field, param.Type, param.Description)
			}),
		})
	}

	mdFile, err := os.OpenFile("tools/fakedata/docs.md", os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}

	groupedData := lo.GroupBy(rows, func(item MdTableRow) string {
		return item.Group
	})

	lo.ForEach(lo.Keys(groupedData), func(item string, _ int) {
		rows := groupedData[item]
		if _, err = fmt.Fprintf(mdFile, "### %s\n", item); err != nil {
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
				li(row.Params),
				f(row.Example),
			); err != nil {
				panic(err)
			}
		}

		if _, err = mdFile.WriteString("\n\n"); err != nil {
			panic(err)
		}
	})
}

func f(lines string) string {
	return strings.ReplaceAll(lines, "\n", "<br>")
}

func li(lines []string) string {
	return strings.Join(lo.Map(lines, func(item string, _ int) string {
		return strings.ReplaceAll(item, "\n", ".")
	}), "<br>")
}
