package ping

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	probing "github.com/prometheus-community/pro-bing"
	"github.com/rivo/tview"
)

var (
	ipAddresses []string
	activeIPs   = make(map[string]int)
	mu          sync.Mutex
)

func InatailizePing(app *tview.Application) {

}

func ParseIPs(input string) []string {
	var ips []string
	for _, ip := range strings.Split(input, ",") {
		ips = append(ips, strings.TrimSpace(ip))
	}
	return ips
}

// func StartPing(ipAddresses []string, monitoringType string) {
// 	var wg sync.WaitGroup

// 	for _, ipAddress := range ipAddresses {
// 		wg.Add(1)
// 		go func(ip string) {
// 			defer wg.Done()
// 			continuousPingIP(ip)
// 		}(ipAddress)
// 	}

// 	wg.Wait()
// }

// func pingIP(ipAddress string) {
// 	pinger, err := probing.NewPinger(ipAddress)
// 	if err != nil {
// 		log.Printf("Error creating pinger for %s: %v\n", ipAddress, err)
// 		return
// 	}

// 	pinger.SetPrivileged(true)
// 	pinger.Count = 3

// 	fmt.Printf("Pinging %s...\n", ipAddress)
// 	err = pinger.Run()
// 	if err != nil {
// 		log.Printf("Error running pinger for %s: %v\n", ipAddress, err)
// 		return
// 	}

// 	stats := pinger.Statistics()
// 	fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
// 	fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
// 		stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
// 	fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
// 		stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
// }

// func continuousPingIP(ipAddress string) {
// 	pinger, err := probing.NewPinger(ipAddress)
// 	if err != nil {
// 		log.Printf("Error creating pinger for %s: %v\n", ipAddress, err)
// 		return
// 	}

// 	pinger.SetPrivileged(true)
// 	pinger.Count = 1

// 	for {
// 		// Run the pinger
// 		err = pinger.Run()
// 		if err != nil {
// 			log.Printf("Error running pinger for %s: %v\n", ipAddress, err)
// 			return
// 		}

// 		// Get statistics
// 		stats := pinger.Statistics()
// 		responseTime := stats.AvgRtt.Seconds() * 1000 // Convert to milliseconds

// 		// updateGraph(ipAddress, responseTime)

// 		// Wait for 1 second before the next ping
// 		time.Sleep(1 * time.Second)
// 	}
// }

func pingResult(ip string) float64 {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		log.Printf("Error creating pinger for %s: %v\n", ip, err)
		return -1
	}

	pinger.SetPrivileged(true)
	pinger.Count = 1

	err = pinger.Run()
	if err != nil {
		log.Printf("Error running pinger for %s: %v\n", ip, err)
		return -1
	}

	stats := pinger.Statistics()
	return stats.AvgRtt.Seconds() * 1000
}

func ipResponseStatus(responseTime float64) (string, tcell.Color) {
	if responseTime < 0 {
		return "[grey]✖ ", tcell.ColorGrey // Indicate failure
	}
	if responseTime < 100 {
		return "[green]▲ ", tcell.ColorGreen
	} else if responseTime < 300 {
		return "[yellow]▲ ", tcell.ColorYellow
	}
	return "[red]▼ ", tcell.ColorRed
}

func PingIP(app *tview.Application, ip string, rowId int, resultTable *tview.Table) {
	mu.Lock()
	activeIPs[ip] = rowId
	mu.Unlock()
	responseTime := pingResult(ip)

	// status := " "
	// color := tcell.ColorWhite

	// // Determine the status based on response time
	// if responseTime < 100 {
	// 	status = "[green]▲ " // Good response
	// 	color = tcell.ColorGreen
	// } else if responseTime < 300 {
	// 	status = "[yellow]▲ " // Moderate response
	// 	color = tcell.ColorYellow
	// } else {
	// 	status = "[red]▼ " // Bad response
	// 	color = tcell.ColorRed
	// }

	status, color := ipResponseStatus(responseTime)

	// Combine status and response time into a single string
	resStatus := fmt.Sprintf("%s: %.2f ms", ip, responseTime)
	if responseTime < 0 {
		resStatus = fmt.Sprintf("%s: Failed", ip)
	}

	log.Printf("Initial ping for %s: %s", ip, resStatus)

	app.QueueUpdateDraw(func() {
		resultTable.SetCell(rowId, 0, tview.NewTableCell(ip).SetTextColor(tview.Styles.PrimaryTextColor).SetAlign(tview.AlignCenter))
		resultTable.SetCell(rowId, 1, tview.NewTableCell(status+resStatus).SetTextColor(color).SetAlign(tview.AlignCenter)) // Combine status with resStatus
	})
	app.Draw()
}

func RefreshPings(app *tview.Application, resultTable *tview.Table) {
	mu.Lock()
	defer mu.Unlock()

	log.Println("Refreshing pings for active IPs:", activeIPs)
	// if resultTable.GetRowCount() <= 1 {
	// 	return
	// }
	// for row := 1; row < resultTable.GetRowCount(); row++ {
	// 	ipCell := resultTable.GetCell(row, 0)
	// 	if ipCell == nil {
	// 		continue
	// 	}
	// 	ip := strings.TrimSpace(ipCell.Text)
	// 	if _, ok := activeIPs[ip]; !ok {
	// 		continue
	// 	}
	// 	// ip := resultTable.GetCell(ipIndex, 0)
	// 	responseTime := pingResult(ip)
	// 	status, color := ipResponseStatus(responseTime)
	// 	resStatus := fmt.Sprintf("%s: %.2f ms", ip, responseTime)
	// 	if responseTime < 0 {
	// 		resStatus = fmt.Sprintf("%s: Failed", ip)
	// 	}

	// 	app.QueueUpdateDraw(func() {
	// 		resultTable.SetCell(row, 1, tview.NewTableCell(status+resStatus).SetTextColor(color).SetAlign(tview.AlignCenter))
	// 	})
	// }
	for ip, rowId := range activeIPs {
		responseTime := pingResult(ip)
		status, color := ipResponseStatus(responseTime)
		resStatus := fmt.Sprintf("%.2f ms", responseTime)
		if responseTime < 0 {
			resStatus = "Failed"
		}

		log.Printf("Refresh ping for %s (row %d): %s", ip, rowId, resStatus)

		app.QueueUpdateDraw(func() {
			resultTable.SetCell(rowId, 1, tview.NewTableCell(status+resStatus).SetTextColor(color).SetAlign(tview.AlignCenter))
		})
		app.Draw()
	}

}

func StopPinging() {
	mu.Lock()
	defer mu.Unlock()
	activeIPs = make(map[string]int)
	log.Println("Stopped pinging, cleared active IPs")
}
