package tracert

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/rivo/tview"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// IPInfo holds geolocation data from ipinfo.io
type IPInfo struct {
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
}

// Tracer holds the configuration for a traceroute operation
type Tracer struct {
	DestIP      string
	Privileged  bool // Only affects non-Windows OSes
	MaxHops     int
	Timeout     time.Duration
	Probes      int // Number of probes per TTL (non-Windows only)
	resultView  *tview.TextView
	app         *tview.Application
	traceText   strings.Builder
	hops        []Hop
}

// Hop represents a single hop in the traceroute
type Hop struct {
	TTL      int
	IP       string
	RTT      float64
	Location string
	Timeout  bool
}

// NewTracer creates a new Tracer instance
func NewTracer(destIP string, app *tview.Application, resultView *tview.TextView) (*Tracer, error) {
	if net.ParseIP(destIP) == nil {
		return nil, fmt.Errorf("invalid destination IP: %s", destIP)
	}
	return &Tracer{
		DestIP:     destIP,
		Privileged: true,
		MaxHops:    30,
		Timeout:    5 * time.Second,
		Probes:     3,
		app:        app,
		resultView: resultView,
	}, nil
}

// SetPrivileged sets whether to use privileged mode (only affects non-Windows)
func (t *Tracer) SetPrivileged(privileged bool) {
	t.Privileged = privileged
}

// Run executes the traceroute and updates the TUI
func (t *Tracer) Run() error {
	t.resultView.Clear()
	t.resultView.SetText(fmt.Sprintf("Traceroute to %s...\n", t.DestIP))
	t.traceText.Reset()
	t.traceText.WriteString(fmt.Sprintf("Traceroute to %s:\n", t.DestIP))
	t.traceText.WriteString("--------------------------------------------------\n")

	if runtime.GOOS == "windows" {
		return t.runWindows()
	}
	return t.runNonWindows()
}

// runWindows performs a traceroute using native tracert on Windows
func (t *Tracer) runWindows() error {
	cmd := exec.Command("tracert", "-d", t.DestIP) // -d avoids DNS lookups
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.updateText(fmt.Sprintf("Failed to get stdout pipe: %v", err))
		return err
	}
	if err := cmd.Start(); err != nil {
		t.updateText(fmt.Sprintf("Failed to start tracert: %v", err))
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		hop, err := parseTracertLine(line)
		if err == nil {
			location, err := getIPLocation(hop.IP)
			if err != nil {
				log.Printf("Location fetch error for %s: %v", hop.IP, err)
			}
			t.addHop(hop.TTL, hop.IP, fmt.Sprintf("%.2f ms", hop.RTT), hop.Timeout, location)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		log.Printf("Tracert command failed: %v", err)
	}
	t.updateText(t.traceText.String())
	return nil
}

// parseTracertLine parses a line from tracert output
func parseTracertLine(line string) (Hop, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 || !isHopLine(fields[0]) {
		return Hop{}, fmt.Errorf("not a hop line")
	}

	var ttl int
	fmt.Sscanf(fields[0], "%d", &ttl)
	if ttl == 0 {
		return Hop{}, fmt.Errorf("invalid TTL")
	}

	ip := fields[len(fields)-1]
	if ip == "*" {
		return Hop{TTL: ttl, IP: "*", Timeout: true}, nil
	}

	rttStr := fields[1]
	if rttStr == "*" {
		return Hop{TTL: ttl, IP: ip, Timeout: true}, nil
	}

	var rtt float64
	fmt.Sscanf(rttStr, "%f", &rtt)
	return Hop{TTL: ttl, IP: ip, RTT: rtt, Timeout: false}, nil
}

// isHopLine checks if a line starts with a number (indicating a hop)
func isHopLine(firstField string) bool {
	_, err := fmt.Sscanf(firstField, "%d", new(int))
	return err == nil
}

// runNonWindows performs an ICMP-based traceroute on non-Windows OSes
func (t *Tracer) runNonWindows() error {
	if !t.Privileged {
		t.updateText("Unprivileged mode not implemented; run with sudo for ICMP")
		return fmt.Errorf("unprivileged mode not implemented; run with sudo for ICMP")
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		t.updateText(fmt.Sprintf("Failed to listen for ICMP: %v (run with admin privileges?)", err))
		return err
	}
	defer conn.Close()

	for ttl := 1; ttl <= t.MaxHops; ttl++ {
		err := conn.IPv4PacketConn().SetTTL(ttl)
		if err != nil {
			t.updateText(fmt.Sprintf("Failed to set TTL: %v", err))
			return err
		}

		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{
				ID:   12345,
				Seq:  ttl,
				Data: []byte("IPmaster"),
			},
		}
		msgBytes, err := msg.Marshal(nil)
		if err != nil {
			t.updateText(fmt.Sprintf("Failed to marshal ICMP message: %v", err))
			return err
		}

		start := time.Now()
		_, err = conn.WriteTo(msgBytes, &net.IPAddr{IP: net.ParseIP(t.DestIP)})
		if err != nil {
			t.updateText(fmt.Sprintf("Failed to send ICMP packet: %v", err))
			return err
		}

		conn.SetReadDeadline(time.Now().Add(t.Timeout))
		reply := make([]byte, 1500)
		_, peer, err := conn.ReadFrom(reply)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				t.addHop(ttl, "*", "N/A", true, "N/A")
				continue
			}
			t.updateText(fmt.Sprintf("Failed to read ICMP reply: %v", err))
			return err
		}

		rtt := time.Since(start).Seconds() * 1000
		ip := peer.(*net.IPAddr).IP.String()
		location, err := getIPLocation(ip)
		if err != nil {
			log.Printf("Location fetch error for %s: %v", ip, err)
		}
		t.addHop(ttl, ip, fmt.Sprintf("%.2f ms", rtt), false, location)

		if ip == t.DestIP {
			break
		}
	}

	t.updateText(t.traceText.String())
	return nil
}

// addHop adds a hop to the trace text and updates the TUI
func (t *Tracer) addHop(ttl int, ip, rtt string, timeout bool, location string) {
	hop := Hop{TTL: ttl, IP: ip, Location: location, Timeout: timeout}
	if !timeout {
		fmt.Sscanf(rtt, "%f ms", &hop.RTT)
	}
	t.hops = append(t.hops, hop)
	t.traceText.WriteString(fmt.Sprintf("Hop %2d: %s (%s) - %s\n", ttl, ip, location, rtt))
	t.updateText(t.traceText.String())
}

// updateText updates the TextView with the current trace text
func (t *Tracer) updateText(text string) {
	t.app.QueueUpdateDraw(func() {
		t.resultView.SetText(text)
	})
	t.app.Draw()
}

// getIPLocation fetches geolocation data for an IP using ipinfo.io
func getIPLocation(ip string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://ipinfo.io/%s/json", ip))
	if err != nil {
		return "Unknown", fmt.Errorf("failed to fetch location for %s: %v", ip, err)
	}
	defer resp.Body.Close()

	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "Unknown", fmt.Errorf("failed to decode location for %s: %v", ip, err)
	}

	if info.City == "" && info.Region == "" && info.Country == "" {
		return "Unknown", nil
	}
	return fmt.Sprintf("%s, %s, %s", info.City, info.Region, info.Country), nil
}