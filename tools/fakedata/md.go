package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/log"

	"github.com/samber/lo"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/evg4b/uncors/pkg/fakedata"
)

type MdTableRow struct {
	Type        string
	Description string
	Options     []string
	Example     string
	Group       string
	Output      string
}

//nolint:cyclop
func generateMdData() {
	rows := make([]MdTableRow, 0)
	for _, typeKey := range fakedata.GetTypes() {
		info := gofakeit.GetFuncLookup(typeKey)
		if info == nil {
			if typeKey != "array" && typeKey != "object" {
				log.Warnf("Type %s not found in fakedata", typeKey)
			}

			continue
		}

		rows = append(rows, MdTableRow{
			Type:        typeKey,
			Description: info.Description,
			Example:     info.Example,
			Group:       info.Category,
			Output:      info.Output,
			Options: lo.Map(info.Params, func(param gofakeit.Param, _ int) string {
				return fmt.Sprintf("%s (%s) - %s", param.Field, param.Type, param.Description)
			}),
		})
	}

	mdFile, err := os.OpenFile("tools/fakedata/docs.md", os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}

	groupedData := lo.GroupBy(rows, func(item MdTableRow) string {
		return item.Group
	})

	groupKeys := lo.Keys(groupedData)

	sort.Strings(groupKeys)

	lo.ForEach(groupKeys, func(item string, _ int) {
		rows := groupedData[item]

		if _, err = fmt.Fprintf(mdFile, "#### %s\n", capitalizeFirstLetter(item)); err != nil {
			panic(err)
		}

		if _, err = mdFile.WriteString("| Type | Description | Options | Return Type | Example |\n"); err != nil {
			panic(err)
		}

		if _, err = mdFile.WriteString("| ---- | ----------- | ------- | ------- | ------- |\n"); err != nil {
			panic(err)
		}

		data := lo.Map(rows, func(item MdTableRow, _ int) string {
			return item.Type
		})

		sort.Strings(data)

		for _, typeStr := range data {
			row, ok := lo.Find(rows, func(item MdTableRow) bool {
				return item.Type == typeStr
			})

			if !ok {
				log.Warnf("Type %s not found in fakedata", typeStr)

				continue
			}

			if _, err = fmt.Fprintf(
				mdFile,
				"| %s | %s | %s | %s | %s |\n",
				process(row.Type),
				process(row.Description),
				processLi(row.Options),
				process(row.Output),
				process(row.Example),
			); err != nil {
				panic(err)
			}
		}

		if _, err = mdFile.WriteString("\n\n"); err != nil {
			panic(err)
		}

		log.Infof("Generated faked data for %s", item)
	})
}

func process(lines string) string {
	return strings.ReplaceAll(lines, "\n", "<br>")
}

func processLi(lines []string) string {
	if len(lines) == 0 {
		return "-"
	}

	return strings.Join(lo.Map(lines, func(item string, _ int) string {
		return strings.ReplaceAll(item, "\n", ".")
	}), "<br>")
}

func capitalizeFirstLetter(str string) string {
	if str == "" {
		return str
	}
	// Convert first character to uppercase and append the rest of the string
	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}
