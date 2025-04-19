package iptables

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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

// func (ipt *IPTables) ShowRoutingTable() error {
// 	ipt.resultView.Clear()
// 	ipt.resultView.SetText("Fetching IP routing table...\n")

// 	var cmd *exec.Cmd
// 	var output strings.Builder

// 	if runtime.GOOS == "windows" {
// 		cmd = exec.Command("route", "print")
// 	} else {
// 		cmd = exec.Command("ip", "route")
// 	}

// 	stdout, err := cmd.StdoutPipe()
// 	if err != nil {
// 		ipt.updateText(fmt.Sprintf("Failed to get stdout pipe: %v", err))
// 		return err
// 	}
// 	if err := cmd.Start(); err != nil {
// 		ipt.updateText(fmt.Sprintf("Failed to start command: %v", err))
// 		return err
// 	}

// 	scanner := bufio.NewScanner(stdout)
// 	output.WriteString("IP Routing Table:\n")
// 	output.WriteString("--------------------------------------------------\n")
// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		if runtime.GOOS == "windows" && strings.Contains(line, "Active Routes") {
// 			continue // Skip header until routes start
// 		}
// 		if strings.TrimSpace(line) != "" {
// 			output.WriteString(line + "\n")
// 		}
// 	}

// 	if err := scanner.Err(); err != nil {
// 		log.Printf("Scanner error: %v", err)
// 	}
// 	if err := cmd.Wait(); err != nil {
// 		log.Printf("Command failed: %v", err)
// 	}

// 	ipt.updateText(output.String())
// 	return nil
// }

func (ipt *IPTables) ShowRoutingTable() error {
	ipt.resultView.Clear()
	ipt.resultView.SetText("Fetching IP routing table...\n")

	var output strings.Builder
	output.WriteString("IP Routing Table:\n")
	output.WriteString("--------------------------------------------------\n")

	// Try exec.Command first
	if runtime.GOOS == "windows" {
		if err := ipt.tryExecCommand("route", []string{"print"}, &output); err != nil {
			// No fallback on Windows; just show error
			ipt.updateText(fmt.Sprintf("Failed to fetch routing table: %v\nNo fallback available on Windows.", err))
			return err
		}
	} else {
		// Linux-based systems (including BusyBox, Alpine)
		if err := ipt.tryExecCommand("ip", []string{"route"}, &output); err != nil {
			log.Printf("ip route command failed: %v, attempting fallback", err)
			if err := ipt.parseProcNetRoute(&output); err != nil {
				ipt.updateText(fmt.Sprintf("Failed to fetch routing table: %v\nFallback (/proc/net/route) also failed: %v", err, err))
				return err
			}
		}
	}

	ipt.updateText(output.String())
	return nil
}

func (ipt *IPTables) tryExecCommand(cmdName string, args []string, output *strings.Builder) error {
	cmd := exec.Command(cmdName, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %v", cmdName, err)
	}

	scanner := bufio.NewScanner(stdout)
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
		return fmt.Errorf("%s command failed: %v", cmdName, err)
	}
	return nil
}

func (ipt *IPTables) parseProcNetRoute(output *strings.Builder) error {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return fmt.Errorf("failed to open /proc/net/route: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Skip header line
	if !scanner.Scan() {
		return fmt.Errorf("/proc/net/route is empty")
	}

	output.WriteString("Interface  Destination       Gateway           Genmask\n")
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue // Skip malformed lines
		}

		iface := fields[0]
		destHex := fields[1]
		gatewayHex := fields[2]
		maskHex := fields[7]

		// Convert hex to IP addresses
		destIP, _ := hexToIP(destHex)
		gatewayIP, _ := hexToIP(gatewayHex)
		genmask, _ := hexToIP(maskHex)

		output.WriteString(fmt.Sprintf("%-10s %-16s %-16s %-16s\n", iface, destIP, gatewayIP, genmask))
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading /proc/net/route: %v", err)
	}
	return nil
}

func hexToIP(hexStr string) (string, error) {
	if len(hexStr) != 8 {
		return "0.0.0.0", fmt.Errorf("invalid hex length: %s", hexStr)
	}
	bytes, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return "0.0.0.0", fmt.Errorf("invalid hex: %s", hexStr)
	}
	ip := net.IPv4(byte(bytes&0xFF), byte((bytes>>8)&0xFF), byte((bytes>>16)&0xFF), byte((bytes>>24)&0xFF))
	return ip.String(), nil
}

func (ipt *IPTables) updateText(text string) {
	ipt.app.QueueUpdateDraw(func() {
		ipt.resultView.SetText(text)
	})
	ipt.app.Draw()
}
