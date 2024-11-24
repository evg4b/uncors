package styles

var (
	DebugBlockStyle = paddedStyle.
			Background(debugColor).
			Foreground(contrastColor)
	WarningBlockStyle = paddedStyle.
				Background(warningColor).
				Foreground(contrastColor)
	InfoBlockStyle = paddedStyle.
			Background(infoColor).
			Foreground(contrastColor)
	ErrorBlockStyle = paddedStyle.
			Background(errorColor).
			Foreground(contrastColor)
)
