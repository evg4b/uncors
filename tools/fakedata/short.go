package main

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/samber/lo"
)

//nolint:cyclop
func generateShortDocs() {
	rows := loadMdData()

	mdFile, err := os.OpenFile("docs.md", os.O_TRUNC|os.O_WRONLY|os.O_CREATE, os.ModePerm)
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

		h(mdFile, h4, capitalizeFirstLetter(item))

		slices.SortFunc(rows, func(a MdTableRow, b MdTableRow) int {
			return strings.Compare(a.Type, b.Type)
		})

		table(
			mdFile,
			[]string{
				"Type",
				"Description",
				"Options",
			},
			lo.Map(rows, func(row MdTableRow, _ int) []string {
				return []string{
					fmt.Sprintf(
						"[%s](https://github.com/evg4b/uncors/tree/main/pkg/fakedata#%s)",
						process(row.Type),
						strings.Trim(strings.ToLower(row.Type), " \n\t"),
					),
					process(row.Description),
					processLi(row.Options),
				}
			}),
		)

		log.Infof("Generated faked data for %s", item)
	})
}
