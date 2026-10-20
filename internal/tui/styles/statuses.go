package styles

import "charm.land/lipgloss/v2"

var (
	PendingTextStyle  = lipgloss.NewStyle().Foreground(DebugColor)
	PendingBlockStyle = PaddedStyle.
				Background(DebugColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	HTTPStatus1xxTextStyle  = underlineStyle
	HTTPStatus1xxBlockStyle = PaddedStyle.
				Background(httpStatus1xxColor).
				Foreground(ContrastColor)

	HTTPStatus2xxTextStyle  = underlineStyle
	HTTPStatus2xxBlockStyle = PaddedStyle.
				Background(httpStatus2xxColor).
				Foreground(ContrastColor)

	HTTPStatus3xxTextStyle  = underlineStyle
	HTTPStatus3xxBlockStyle = PaddedStyle.
				Background(httpStatus3xxColor).
				Foreground(ContrastColor)

	HTTPStatus4xxTextStyle  = underlineStyle
	HTTPStatus4xxBlockStyle = PaddedStyle.
				Background(httpStatus4xxColor).
				Foreground(ContrastColor)

	HTTPStatus5xxTextStyle  = underlineStyle
	HTTPStatus5xxBlockStyle = PaddedStyle.
				Background(httpStatus5xxColor).
				Foreground(ContrastColor)

	HTTPStatusOtherTextStyle  = underlineStyle
	HTTPStatusOtherBlockStyle = PaddedStyle.
					Background(httpStatusOtherColor).
					Foreground(ContrastColor)

	HTTPStatusCancelledTextStyle  = underlineStyle
	HTTPStatusCancelledBlockStyle = PaddedStyle.
					Background(httpStatusCancelledColor).
					Foreground(ContrastColor)
)
