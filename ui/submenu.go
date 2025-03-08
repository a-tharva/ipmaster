package ui

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/a-tharva/ipmaster/ipinfo"
	"github.com/a-tharva/ipmaster/ping"
	"github.com/a-tharva/ipmaster/tracert"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	pingStop chan struct{} // Channel to signal stopping the ping
)

func showIPInfo(app *tview.Application) {
	infoView := tview.NewTextView().
		SetText("IP Info Page").SetTextAlign(tview.AlignCenter)

	ipViewTable := tview.NewTable()

	headers := []string{"Interface Name", "MTU", "Flags", "IP Addresses"}
	for i, header := range headers {
		ipViewTable.SetCell(0, i,
			tview.NewTableCell(header).SetTextColor(tview.Styles.SecondaryTextColor).SetAlign(tview.AlignCenter))
	}

	ifaceDetails, err := ipinfo.GetIpDetails()

	if err != nil {
		infoView.SetText(fmt.Sprintf("Error fetching IP details: %v", err))
	} else if len(ifaceDetails) == 0 {
		infoView.SetText("No network interfaces found.")
	} else {
		for i, detail := range ifaceDetails {
			ipViewTable.SetCell(i+1, 0, tview.NewTableCell(detail.Name))
			ipViewTable.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprintf("%d", detail.MTU)))
			ipViewTable.SetCell(i+1, 2, tview.NewTableCell(detail.Flags.String()))
			ipViewTable.SetCell(i+1, 3, tview.NewTableCell(fmt.Sprintf("%v", detail.IPs)))
		}
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(infoView, 0, 1, true).
		AddItem(ipViewTable, 0, 2, true)

	app.SetRoot(flex, true)

	setBackCapture(app)
}

func showTracert(app *tview.Application) {
	// Placeholder for tracert (to be implemented)
	tracertView := tview.NewTextView().
		SetText("Tracert Page").SetTextAlign(tview.AlignCenter)

	inputField := tview.NewInputField().
		SetLabel("Enter destination IP: ").
		SetFieldWidth(0)

	resultView := tview.NewTextView().
		SetLabel("Enter an IP to see the traceroute path...").
		SetWordWrap(true)

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			destIP := strings.TrimSpace(inputField.GetText())
			if net.ParseIP(destIP) == nil {
				inputField.SetFieldBackgroundColor(tcell.ColorRed)
				inputField.SetLabel(fmt.Sprintf("Invalid IP: %s ", destIP))
				return
			}

			inputField.SetFieldBackgroundColor(tcell.ColorBlue)
			inputField.SetLabel("Enter destination IP: ")

			resultView.SetText(fmt.Sprintf("Tracing route to %s...", destIP))

			//Create and configure Traacer
			tracer, err := tracert.NewTracer(destIP, app, resultView)
			if err != nil {
				resultView.SetText(fmt.Sprintf("Traceroute to %s failed: %v", destIP, err))
				return
			}
			// SetPrivileged(true) is default; only affects non-Windows
			go func() {
				err := tracer.Run()
				if err != nil {
					app.QueueUpdateDraw(func() {
						resultView.SetText(fmt.Sprintf("Traceroute to %s failed: %v", destIP, err))
					})
				}
			}()
		}
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tracertView, 0, 1, true).
		AddItem(inputField, 1, 1, true).
		AddItem(resultView, 0, 5, true)

	app.SetRoot(flex, true)
	app.SetFocus(inputField)
	setBackCapture(app)
}

func showPing(app *tview.Application) {
	pingView := tview.NewTextView().
		SetText("Ping Page").SetTextAlign(tview.AlignCenter)

	inputField := tview.NewInputField().
		SetLabel("Enter IPs (comma-separated): ").
		SetFieldWidth(0)

	resultTable := tview.NewTable().SetBorders(true)
	headers := []string{"IP Address", "Status"}
	for i, header := range headers {
		resultTable.SetCell(0, i,
			tview.NewTableCell(header).SetTextColor(tview.Styles.SecondaryTextColor).SetAlign(tview.AlignCenter))
	}

	var ipPresent bool
	ticker := time.NewTicker(2 * time.Second)
	pingStop = make(chan struct{})

	// Handle Enter key press to trigger the ping
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ipAddresses := strings.Split(inputField.GetText(), ",")
			resultTable.Clear()
			for i, header := range headers {
				resultTable.SetCell(0, i,
					tview.NewTableCell(header).SetTextColor(tview.Styles.SecondaryTextColor).SetAlign(tview.AlignCenter))
			}

			row := 1
			for _, ip := range ipAddresses {
				ip = strings.TrimSpace(ip)
				if net.ParseIP(ip) == nil {
					inputField.SetFieldBackgroundColor(tcell.ColorRed)
					inputField.SetLabel(fmt.Sprintf("Invalid IP: %s ", ip))
					return
				}
				resultTable.SetCell(row, 0, tview.NewTableCell(ip).SetTextColor(tview.Styles.PrimaryTextColor).SetAlign(tview.AlignCenter))
				resultTable.SetCell(row, 1, tview.NewTableCell("Pinging...").SetTextColor(tcell.ColorGrey).SetAlign(tview.AlignCenter))
				go ping.PingIP(app, ip, row, resultTable) // Initial ping
				row++
			}
			inputField.SetFieldBackgroundColor(tcell.ColorBlue)
			inputField.SetLabel("Enter IPs (comma-separated): ")
			// log.Println("Started pinging IPs:", ipAddresses)
			ipPresent = true
		}
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pingView, 0, 1, true).
		AddItem(inputField, 1, 1, true).
		AddItem(resultTable, 0, 5, true)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if ipPresent {
					// log.Println("Ticker fired, refreshing pings...")
					ping.RefreshPings(app, resultTable)
				}
			case <-pingStop:
				// log.Println("Ping stopped by pingStop signal")
				return
			}
		}
	}()

	app.SetRoot(flex, true)
	app.SetFocus(inputField)
	setBackCapture(app)

	// app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	// 	if event.Key() == tcell.KeyEscape {
	// 		stopContinuousPing() // Stop all continuous pings
	// 		Create(app)
	// 		return nil
	// 	}
	// 	return event
	// })
}

func stopContinuousPing() {
	if pingStop != nil {
		close(pingStop)
		pingStop = nil // Reinitialize to avoid panic on reuse
	}
	ping.StopPinging()
	// log.Println("Continuous ping stopped")
}
