package iptables

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"

	"github.com/rivo/tview"
)

type IPTables struct {
	resultView *tview.TextView
	app        *tview.Application
}

func NewIPTables(app *tview.Application, resultView *tview.TextView) *IPTables {
	return &IPTables{
		resultView: resultView,
		app:        app,
	}
}

func (ipt *IPTables) ShowRoutingTable() error {
	ipt.resultView.Clear()
	ipt.resultView.SetText("Fetching IP routing table...\n")

	var cmd *exec.Cmd
	var output strings.Builder

	if runtime.GOOS == "windows" {
		cmd = exec.Command("route", "print")
	} else {
		cmd = exec.Command("ip", "route")
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ipt.updateText(fmt.Sprintf("Failed to get stdout pipe: %v", err))
		return err
	}
	if err := cmd.Start(); err != nil {
		ipt.updateText(fmt.Sprintf("Failed to start command: %v", err))
		return err
	}

	scanner := bufio.NewScanner(stdout)
	output.WriteString("IP Routing Table:\n")
	output.WriteString("--------------------------------------------------\n")
	for scanner.Scan() {
		line := scanner.Text()
		if runtime.GOOS == "windows" && strings.Contains(line, "Active Routes") {
			continue // Skip header until routes start
		}
		if strings.TrimSpace(line) != "" {
			output.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		log.Printf("Command failed: %v", err)
	}

	ipt.updateText(output.String())
	return nil
}

func (ipt *IPTables) updateText(text string) {
	ipt.app.QueueUpdateDraw(func() {
		ipt.resultView.SetText(text)
	})
	ipt.app.Draw()
}
