package render

import (
	"fmt"
	"io"

	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/spf13/pflag"
)

const unLetters = `██    ██ ███    ██ 
██    ██ ████   ██ 
██    ██ ██ ██  ██ 
██    ██ ██  ██ ██ 
 ██████  ██   ████ `

const corsLetters = ` ██████  ██████  ██████  ███████
██      ██    ██ ██   ██ ██     
██      ██    ██ ██████  ███████
██      ██    ██ ██   ██      ██
 ██████  ██████  ██   ██ ███████`

func Logo(theme *styles.Theme, version string) string {
	return lipgloss.JoinVertical(
		lipgloss.Right,
		lipgloss.JoinHorizontal(lipgloss.Top, theme.LogoRed.Render(unLetters), theme.LogoYellow.Render(corsLetters)),
		fmt.Sprintf("version: %s", version),
	)
}

func PrintLogo(out io.Writer, theme *styles.Theme, version string) {
	_, err := fmt.Fprintln(out, Logo(theme, version))
	if err != nil {
		panic(err)
	}
}

func PrintUsage(flags *pflag.FlagSet, version string) {
	out := flags.Output()

	PrintLogo(out, styles.Default, version)

	_, err := fmt.Fprintf(out, "\n%s\n", flags.FlagUsages())
	if err != nil {
		panic(err)
	}
}
