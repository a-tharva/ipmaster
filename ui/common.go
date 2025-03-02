package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Function to update the style of the selected cell
func updateSelectedStyle(table *tview.Table, selectedRow int) {
	for i := 0; i < len(ipOptions); i++ {
		if i == selectedRow {
			table.GetCell(i, 0).SetTextColor(tcell.ColorWhite) // Highlight selected
		} else {
			table.GetCell(i, 0).SetTextColor(tcell.ColorGrey) // Default style
		}
	}
}

func setBackCapture(app *tview.Application) {
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case 'b', tcell.KeyEscape:
			stopContinuousPing()
			Create(app)
			return nil
		}
		return event
	})
}
