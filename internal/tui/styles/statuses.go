package styles

var (
	HTTPStatus1xxTextStyle  = underlineStyle
	HTTPStatus1xxBlockStyle = paddedStyle.
				Background(httpStatus1xxColor).
				Foreground(contrastColor)

	HTTPStatus2xxTextStyle  = underlineStyle
	HTTPStatus2xxBlockStyle = paddedStyle.
				Background(httpStatus2xxColor).
				Foreground(contrastColor)

	HTTPStatus3xxTextStyle  = underlineStyle
	HTTPStatus3xxBlockStyle = paddedStyle.
				Background(httpStatus3xxColor).
				Foreground(contrastColor)

	HTTPStatus4xxTextStyle  = underlineStyle
	HTTPStatus4xxBlockStyle = paddedStyle.
				Background(httpStatus4xxColor).
				Foreground(contrastColor)

	HTTPStatus5xxTextStyle  = underlineStyle
	HTTPStatus5xxBlockStyle = paddedStyle.
				Background(httpStatus5xxColor).
				Foreground(contrastColor)
)
