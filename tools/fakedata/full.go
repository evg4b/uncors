package main

import (
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/samber/lo"
)

//nolint:cyclop
func generateFullDocs() {
	rows := loadMdData()

	mdFile, err := os.OpenFile("../../pkg/fakedata/README.md", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}

	defer mdFile.Close()

	groupedData := lo.GroupBy(rows, func(item MdTableRow) string {
		return item.Group
	})

	groupKeys := lo.Keys(groupedData)

	sort.Strings(groupKeys)

	h(mdFile, h1, "github.com/evg4b/uncors/pkg/fakedata")

	lo.ForEach(groupKeys, func(item string, _ int) {
		rows := groupedData[item]
		h(mdFile, h2, capitalizeFirstLetter(item))

		slices.SortFunc(rows, func(a MdTableRow, b MdTableRow) int {
			return strings.Compare(a.Type, b.Type)
		})

		lo.ForEach(rows, func(row MdTableRow, _ int) {
			h(mdFile, h3, capitalizeFirstLetter(row.Type))
			p(mdFile, row.Description)
			p(mdFile, "Return type: "+"`"+row.Output+"`")
			if len(row.Options) > 1 {
				p(mdFile, "Options:")
				li(mdFile, row.Options)
			}
			p(mdFile, "Example:")
			code(mdFile, row.Example)
		})

		log.Infof("Generated faked data for %s", item)
	})
}
