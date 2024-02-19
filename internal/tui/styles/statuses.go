package styles

import "github.com/charmbracelet/lipgloss"

type StatusStyle struct {
	BlockStyle         lipgloss.Style
	MainTextStyle      lipgloss.Style
	SecondaryTextStyle lipgloss.Style
}

var InformationalStyle = StatusStyle{
	BlockStyle:         InfoBlock.Copy(),
	MainTextStyle:      InfoText.Copy(),
	SecondaryTextStyle: DisabledText.Copy(),
}

var SuccessStyle = StatusStyle{
	BlockStyle:         SuccessBlock.Copy(),
	MainTextStyle:      SuccessText.Copy(),
	SecondaryTextStyle: DisabledText.Copy(),
}

var RedirectionStyle = StatusStyle{
	BlockStyle:         WarningBlock.Copy(),
	MainTextStyle:      WarningText.Copy(),
	SecondaryTextStyle: DisabledText.Copy(),
}

var ClientErrorStyle = StatusStyle{
	BlockStyle:         ErrorBlock.Copy(),
	MainTextStyle:      ErrorText.Copy(),
	SecondaryTextStyle: DisabledText.Copy(),
}

var ServerErrorStyle = ClientErrorStyle

var CanceledStyle = StatusStyle{
	BlockStyle:         DisabledBlock.Copy(),
	MainTextStyle:      DisabledText.Copy(),
	SecondaryTextStyle: DisabledText.Copy(),
}

var PendingStyle = StatusStyle{
	BlockStyle:         DisabledBlock.Copy(),
	MainTextStyle:      DisabledText.Copy(),
	SecondaryTextStyle: DisabledText.Copy(),
}
