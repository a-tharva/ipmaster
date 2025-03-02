package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var ipOptions = []string{
	"ip info",
	"tracert",
	"ping",
}

func Start() error {
	app := tview.NewApplication()
	Create(app)
	return app.Run()
}

func Create(app *tview.Application) {
	textView := createTextView("IPmaster")
	table := createTable(ipOptions)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(table, 0, 8, false)

	table.SetSelectedFunc(func(row, column int) {
		handleTableSelection(app, row)
	})

	app.SetRoot(flex, true)
	app.SetInputCapture(createInputCapture(app, table))

}

func createTextView(text string) *tview.TextView {
	return tview.NewTextView().
		SetText(text).
		SetTextAlign(tview.AlignCenter)
}

func createTable(options []string) *tview.Table {
	table := tview.NewTable()
	for i, option := range options {
		cell := tview.NewTableCell(option).
			SetAlign(tview.AlignLeft).
			SetTextColor(tcell.ColorWhite) // Default text color
		table.SetCell(i, 0, cell)
	}
	table.SetBorder(true)
	table.Select(0, 0)            // Start with the first option selected
	updateSelectedStyle(table, 0) // Highlight the first row
	return table
}

func handleTableSelection(app *tview.Application, row int) {
	switch row {
	case 0:
		showIPInfo(app)
	case 1:
		showTracert(app)
	case 2:
		showPing(app)
	}
}

func createInputCapture(app *tview.Application, table *tview.Table) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case 'q':
			app.SetRoot(tview.NewModal().
				SetText("Quit IPmaster?").
				AddButtons([]string{"Yes", "No"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Yes" {
						app.Stop()
					} else {
						Create(app)
					}
				}), true)
			app.Stop()
		case tcell.KeyEnter:
			row, _ := table.GetSelection()
			handleTableSelection(app, row)
		case tcell.KeyUp:
			navigateTable(table, -1)
		case tcell.KeyDown:
			navigateTable(table, 1)
		case tcell.KeyEsc:
			app.Stop()
		}
		return event
	}
}

func navigateTable(table *tview.Table, direction int) {
	row, _ := table.GetSelection()
	newRow := (row + direction + len(ipOptions)) % len(ipOptions) // Wrap around
	table.Select(newRow, 0)
	updateSelectedStyle(table, newRow) // Update style for the new selection
}
