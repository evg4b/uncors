package styles

import "github.com/charmbracelet/lipgloss"

type StatusStyle struct {
	BlockStyle         lipgloss.Style
	MainTextStyle      lipgloss.Style
	SecondaryTextStyle lipgloss.Style
}

var InformationalStyle = StatusStyle{
	BlockStyle:         InfoBlock,
	MainTextStyle:      InfoText,
	SecondaryTextStyle: DisabledText,
}

var SuccessStyle = StatusStyle{
	BlockStyle:         SuccessBlock,
	MainTextStyle:      SuccessText,
	SecondaryTextStyle: DisabledText,
}

var RedirectionStyle = StatusStyle{
	BlockStyle:         WarningBlock,
	MainTextStyle:      WarningText,
	SecondaryTextStyle: DisabledText,
}

var ClientErrorStyle = StatusStyle{
	BlockStyle:         ErrorBlock,
	MainTextStyle:      ErrorText,
	SecondaryTextStyle: DisabledText,
}

var ServerErrorStyle = ClientErrorStyle

var CanceledStyle = StatusStyle{
	BlockStyle:         DisabledBlock,
	MainTextStyle:      DisabledText.Strikethrough(true),
	SecondaryTextStyle: DisabledText.Strikethrough(true),
}

var PendingStyle = StatusStyle{
	BlockStyle:         DisabledBlock,
	MainTextStyle:      DisabledText,
	SecondaryTextStyle: DisabledText,
}
