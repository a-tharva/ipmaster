package ports

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rivo/tview"
)

type PortScanner struct {
	TargetIP   string
	StartPort  int
	EndPort    int
	Timeout    time.Duration
	resultView *tview.TextView
	app        *tview.Application
}

func NewPortScanner(targetIP string, app *tview.Application, resultView *tview.TextView) (*PortScanner, error) {
	if net.ParseIP(targetIP) == nil {
		return nil, fmt.Errorf("invalid target IP: %s", targetIP)
	}
	return &PortScanner{
		TargetIP:   targetIP,
		StartPort:  1,
		EndPort:    1024,
		Timeout:    2 * time.Second,
		resultView: resultView,
		app:        app,
	}, nil
}

func (ps *PortScanner) ScanPorts() error {
	ps.resultView.Clear()
	ps.resultView.SetText(fmt.Sprintf("Scanning ports on %s (%d-%d)...\n", ps.TargetIP, ps.StartPort, ps.EndPort))

	var openPorts []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for port := ps.StartPort; port <= ps.EndPort; port++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			addr := fmt.Sprintf("%s:%d", ps.TargetIP, p)
			conn, err := net.DialTimeout("tcp", addr, ps.Timeout)
			if err == nil {
				mu.Lock()
				openPorts = append(openPorts, strconv.Itoa(p))
				mu.Unlock()
				conn.Close()
			}
		}(port)
	}

	wg.Wait()

	if len(openPorts) == 0 {
		ps.updateText(fmt.Sprintf("No open ports found on %s (%d-%d)", ps.TargetIP, ps.StartPort, ps.EndPort))
	} else {
		ps.updateText(fmt.Sprintf("Open ports on %s: %s", ps.TargetIP, strings.Join(openPorts, ", ")))
	}
	log.Printf("Port scan completed for %s: %v", ps.TargetIP, openPorts)
	return nil
}

func (ps *PortScanner) updateText(text string) {
	ps.app.QueueUpdateDraw(func() {
		ps.resultView.SetText(text)
	})
	ps.app.Draw()
}
