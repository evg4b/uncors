package styles

import "github.com/charmbracelet/lipgloss"

var (
	HTTPStatus1xxBlockStyle = lipgloss.NewStyle().
				Background(httpStatus1xxColor).
				Foreground(contrastColor).
				Padding(0, 1)
	HTTPStatus1xxTextStyle = lipgloss.NewStyle()

	HTTPStatus2xxBlockStyle = lipgloss.NewStyle().
				Background(httpStatus2xxColor).
				Foreground(contrastColor).
				Padding(0, 1)
	HTTPStatus2xxTextStyle = lipgloss.NewStyle()

	HTTPStatus3xxBlockStyle = lipgloss.NewStyle().
				Background(httpStatus3xxColor).
				Foreground(contrastColor).
				Padding(0, 1)
	HTTPStatus3xxTextStyle = lipgloss.NewStyle()

	HTTPStatus4xxBlockStyle = lipgloss.NewStyle().
				Background(httpStatus4xxColor).
				Foreground(contrastColor).
				Padding(0, 1)
	HTTPStatus4xxTextStyle = lipgloss.NewStyle().
				Foreground(httpStatus4xxColor)

	HTTPStatus5xxBlockStyle = lipgloss.NewStyle().
				Background(httpStatus5xxColor).
				Foreground(contrastColor).
				Padding(0, 1)
	HTTPStatus5xxTextStyle = lipgloss.NewStyle().
				Foreground(httpStatus5xxColor)
)
