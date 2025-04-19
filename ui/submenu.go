package ui

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/a-tharva/ipmaster/bgp"
	"github.com/a-tharva/ipmaster/ipinfo"
	"github.com/a-tharva/ipmaster/iptables"
	"github.com/a-tharva/ipmaster/ping"
	"github.com/a-tharva/ipmaster/ports"
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
}

func stopContinuousPing() {
	if pingStop != nil {
		close(pingStop)
		pingStop = nil // Reinitialize to avoid panic on reuse
	}
	ping.StopPinging()
	// log.Println("Continuous ping stopped")
}

func showPorts(app *tview.Application) {
	portsView := tview.NewTextView().
		SetText("Port Scan Page").SetTextAlign(tview.AlignCenter)

	inputField := tview.NewInputField().
		SetLabel("Enter target IP: ").
		SetFieldWidth(0)

	resultView := tview.NewTextView().
		SetText("Enter an IP to scan for open ports...").
		SetWordWrap(true)

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			targetIP := strings.TrimSpace(inputField.GetText())
			if net.ParseIP(targetIP) == nil {
				inputField.SetFieldBackgroundColor(tcell.ColorRed)
				inputField.SetText(fmt.Sprintf("Invalid IP: %s", targetIP))
				return
			}

			inputField.SetFieldBackgroundColor(tcell.ColorBlue)
			resultView.SetText(fmt.Sprintf("Scanning ports on %s...", targetIP))

			scanner, err := ports.NewPortScanner(targetIP, app, resultView)
			if err != nil {
				resultView.SetText(fmt.Sprintf("Port scan failed: %v", err))
				return
			}
			go func() {
				err := scanner.ScanPorts()
				if err != nil {
					app.QueueUpdateDraw(func() {
						resultView.SetText(fmt.Sprintf("Port scan on %s failed: %v", targetIP, err))
					})
				}
			}()
		}
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(portsView, 0, 1, true).
		AddItem(inputField, 1, 1, true).
		AddItem(resultView, 0, 5, true)

	app.SetRoot(flex, true)
	app.SetFocus(inputField)
	setBackCapture(app)
}

func showIPTables(app *tview.Application) {
	iptablesView := tview.NewTextView().
		SetText("IP Tables Page").SetTextAlign(tview.AlignCenter)

	resultView := tview.NewTextView().
		SetText("Fetching IP routing table...").
		SetWordWrap(true)

	ipt := iptables.NewIPTables(app, resultView)
	go func() {
		err := ipt.ShowRoutingTable()
		if err != nil {
			app.QueueUpdateDraw(func() {
				resultView.SetText(fmt.Sprintf("Failed to fetch IP routing table: %v", err))
			})
		}
	}()

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(iptablesView, 0, 1, true).
		AddItem(resultView, 0, 5, true)

	app.SetRoot(flex, true)
	setBackCapture(app)
}

func showBGP(app *tview.Application) {
	bgpView := tview.NewTextView().
		SetText("BGP Routes Page").SetTextAlign(tview.AlignCenter)

	inputField := tview.NewInputField().
		SetLabel("Enter IP prefix (e.g., 8.8.8.0/24): ").
		SetFieldWidth(0)

	resultView := tview.NewTextView().
		SetText("Enter an IP prefix to see BGP routing information...").
		SetWordWrap(true)

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			prefix := strings.TrimSpace(inputField.GetText())
			// Basic prefix validation (e.g., "x.x.x.x/y")
			parts := strings.Split(prefix, "/")
			if len(parts) != 2 || net.ParseIP(parts[0]) == nil {
				inputField.SetFieldBackgroundColor(tcell.ColorRed)
				inputField.SetText(fmt.Sprintf("Invalid prefix: %s", prefix))
				return
			}
			if mask, err := strconv.Atoi(parts[1]); err != nil || mask < 0 || mask > 32 {
				inputField.SetFieldBackgroundColor(tcell.ColorRed)
				inputField.SetText(fmt.Sprintf("Invalid mask: %s", parts[1]))
				return
			}

			inputField.SetFieldBackgroundColor(tcell.ColorBlue)
			resultView.SetText(fmt.Sprintf("Fetching BGP routes for %s...", prefix))

			bgpInstance := bgp.NewBGP(prefix, app, resultView)
			go func() {
				err := bgpInstance.ShowBGPRoutes()
				if err != nil {
					app.QueueUpdateDraw(func() {
						resultView.SetText(fmt.Sprintf("Failed to fetch BGP routes for %s: %v", prefix, err))
					})
				}
			}()
		}
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(bgpView, 0, 1, true).
		AddItem(inputField, 1, 1, true).
		AddItem(resultView, 0, 5, true)

	app.SetRoot(flex, true)
	app.SetFocus(inputField)
	setBackCapture(app)
}
