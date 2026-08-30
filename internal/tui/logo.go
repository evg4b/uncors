package tui

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

var (
	red    = styles.LogoRed.Render
	yellow = styles.LogoYellow.Render
)

func Logo(version string) string {
	return lipgloss.JoinVertical(
		lipgloss.Right,
		lipgloss.JoinHorizontal(lipgloss.Top, red(unLetters), yellow(corsLetters)),
		fmt.Sprintf("version: %s", version),
	)
}

func PrintLogo(out io.Writer, version string) {
	_, err := fmt.Fprintln(out, Logo(version))
	if err != nil {
		panic(err)
	}
}

// PrintUsage renders the standard `--help` output for a flag set: the logo,
// followed by the flag descriptions.
func PrintUsage(flags *pflag.FlagSet, version string) {
	out := flags.Output()

	PrintLogo(out, version)

	_, err := fmt.Fprintf(out, "\n%s\n", flags.FlagUsages())
	if err != nil {
		panic(err)
	}
}
