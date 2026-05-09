package styles

var (
	DebugBlockStyle = PaddedStyle.
			Background(DebugColor).
			Foreground(ContrastColor)
	WarningBlockStyle = PaddedStyle.
				Background(WarningColor).
				Foreground(ContrastColor)
	InfoBlockStyle = PaddedStyle.
			Background(InfoColor).
			Foreground(ContrastColor)
	ErrorBlockStyle = PaddedStyle.
			Background(ErrorColor).
			Foreground(ContrastColor)
)
