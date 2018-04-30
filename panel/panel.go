package panel

import "github.com/gizak/termui"

// logPanel extends a termui List widget
// see NewLogPanel
type logPanel struct {
	Panel    *termui.List
	messages []string
}

// NewLogPanel produces a termui widget for rendering rolling logs
// for use in grid layouts
func NewLogPanel(title string, height int) *logPanel {
	ls := termui.NewList()
	ls.ItemFgColor = termui.ColorYellow
	ls.BorderLabel = title
	ls.Height = height

	return &logPanel{
		Panel:    ls,
		messages: make([]string, 0),
	}
}

// AddMessage adds a message to the list
func (lp *logPanel) AddMessage(msg string) {
	lp.messages = append(lp.messages, msg)

	maxRows := lp.Panel.Height - 2

	if len(lp.messages) > maxRows {
		lp.Panel.Items = lp.messages[(len(lp.messages) - maxRows):]
		return
	}

	lp.Panel.Items = lp.messages
}
