package main

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/samber/lo"
)

const (
	h1 = 1
	h2 = 2
	h3 = 2
	h4 = 4
	h5 = 5
)

type MdTableRow struct {
	Type        string
	Description string
	Options     []string
	Example     string
	Group       string
	Output      string
}

func table(out io.Writer, header []string, rows [][]string) {
	if _, err := fmt.Fprintln(out, "| "+strings.Join(header, " | ")+" |"); err != nil {
		panic(err)
	}
	headerSeparators := lo.Map(header, func(item string, _ int) string {
		return strings.Repeat("-", len(item))
	})
	if _, err := fmt.Fprintln(out, "| "+strings.Join(headerSeparators, " | ")+" |"); err != nil {
		panic(err)
	}
	lo.ForEach(rows, func(cells []string, _ int) {
		if _, err := fmt.Fprintln(out, "| "+strings.Join(cells, " | ")+" |"); err != nil {
			panic(err)
		}
	})
	br(out)
}

func h(out io.Writer, level int, text string) {
	if _, err := fmt.Fprintln(out, strings.Repeat("#", level)+" "+text); err != nil {
		panic(err)
	}
	br(out)
}

func br(out io.Writer) {
	if _, err := fmt.Fprintln(out, ""); err != nil {
		panic(err)
	}
}

func p(out io.Writer, text string) {
	br(out)
	if _, err := fmt.Fprintln(out, text); err != nil {
		panic(err)
	}
	br(out)
}

func li(out io.Writer, items []string) {
	br(out)
	lo.ForEach(items, func(item string, _ int) {
		if _, err := fmt.Fprintln(out, " - "+item); err != nil {
			panic(err)
		}
	})
	br(out)
}

func code(out io.Writer, text string) {
	br(out)
	if _, err := fmt.Fprintln(out, "```\n"+text+"\n```"); err != nil {
		panic(err)
	}
	br(out)
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

func loadMdData() []MdTableRow {
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

	return rows
}
