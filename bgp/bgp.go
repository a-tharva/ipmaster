package bgp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/rivo/tview"
)

// BGPResponse represents the structure of bgpview.io API response
type BGPResponse struct {
	Data struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
			Asns   []struct {
				Asn         int    `json:"asn"`
				Description string `json:"description"`
				CountryCode string `json:"country_code"`
			} `json:"asns"`
		} `json:"prefixes"`
	} `json:"data"`
}

// BGP manages BGP route display
type BGP struct {
	Prefix     string
	resultView *tview.TextView
	app        *tview.Application
}

// NewBGP creates a new BGP instance
func NewBGP(prefix string, app *tview.Application, resultView *tview.TextView) *BGP {
	return &BGP{
		Prefix:     prefix,
		resultView: resultView,
		app:        app,
	}
}

// ShowBGPRoutes displays BGP routing information
func (bgp *BGP) ShowBGPRoutes() error {
	bgp.resultView.Clear()
	bgp.resultView.SetText(fmt.Sprintf("Fetching BGP routes for %s...\n", bgp.Prefix))

	var output strings.Builder
	output.WriteString(fmt.Sprintf("BGP Routes for %s:\n", bgp.Prefix))
	output.WriteString("--------------------------------------------------\n")

	// Try local BGP tools first
	hasBGPData := false
	if runtime.GOOS == "windows" {
		hasBGPData = bgp.tryWindowsRoutePrint(&output)
	} else {
		// Linux-based systems (try birdc)
		hasBGPData = bgp.tryBirdc(&output)
	}

	// If no BGP data found locally, use API fallback
	if !hasBGPData {
		log.Printf("No BGP data found locally, falling back to bgpview.io API")
		if err := bgp.fetchFromBGPView(&output); err != nil {
			bgp.updateText(fmt.Sprintf("Failed to fetch BGP routes: %v\nFallback API also failed: %v", err, err))
			return err
		}
	}

	bgp.updateText(output.String())
	return nil
}

// tryWindowsRoutePrint attempts to fetch BGP routes from route print on Windows
func (bgp *BGP) tryWindowsRoutePrint(output *strings.Builder) bool {
	cmd := exec.Command("route", "print")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to get stdout pipe for route print: %v", err)
		return false
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start route print: %v", err)
		return false
	}

	scanner := bufio.NewScanner(stdout)
	inActiveRoutes := false
	foundPrefix := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.Contains(line, "Active Routes:") {
			inActiveRoutes = true
			continue
		}
		if inActiveRoutes && strings.Contains(line, "Persistent Routes:") {
			break
		}
		if inActiveRoutes {
			fields := strings.Fields(line)
			if len(fields) >= 5 && fields[0] == bgp.Prefix {
				output.WriteString(fmt.Sprintf("Network: %s  Netmask: %s  Gateway: %s  Interface: %s  Metric: %s\n",
					fields[0], fields[1], fields[2], fields[3], fields[4]))
				foundPrefix = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		log.Printf("route print command failed: %v", err)
		return false
	}

	if !foundPrefix {
		output.Reset()
		output.WriteString(fmt.Sprintf("BGP Routes for %s:\n", bgp.Prefix))
		output.WriteString("--------------------------------------------------\n")
		output.WriteString("No BGP-specific routes found in local routing table.\n")
	}
	return foundPrefix
}

// tryBirdc attempts to fetch BGP routes using birdc on Linux/macOS
func (bgp *BGP) tryBirdc(output *strings.Builder) bool {
	cmd := exec.Command("birdc", "show", "route", "for", bgp.Prefix)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to get stdout pipe for birdc: %v", err)
		return false
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start birdc: %v", err)
		return false
	}

	scanner := bufio.NewScanner(stdout)
	foundData := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			output.WriteString(line + "\n")
			foundData = true
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		log.Printf("birdc command failed: %v", err)
		return false
	}

	if !foundData {
		output.Reset()
		output.WriteString(fmt.Sprintf("BGP Routes for %s:\n", bgp.Prefix))
		output.WriteString("--------------------------------------------------\n")
		output.WriteString("No BGP routes found locally (birdc not configured or unavailable).\n")
	}
	return foundData
}

// fetchFromBGPView fetches BGP data from bgpview.io as a fallback
func (bgp *BGP) fetchFromBGPView(output *strings.Builder) error {
	url := fmt.Sprintf("https://api.bgpview.io/prefix/%s", bgp.Prefix)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch BGP data from API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	var bgpResp BGPResponse
	if err := json.NewDecoder(resp.Body).Decode(&bgpResp); err != nil {
		return fmt.Errorf("failed to decode BGP API response: %v", err)
	}

	if len(bgpResp.Data.Prefixes) == 0 {
		output.WriteString("No BGP routes found for this prefix.\n")
		return nil
	}

	output.WriteString("Prefix             ASN    Description                Country\n")
	for _, prefix := range bgpResp.Data.Prefixes {
		for _, asn := range prefix.Asns {
			output.WriteString(fmt.Sprintf("%-18s %-6d %-25s %-2s\n",
				prefix.Prefix, asn.Asn, asn.Description, asn.CountryCode))
		}
	}
	return nil
}

// updateText updates the TextView with the current result
func (bgp *BGP) updateText(text string) {
	bgp.app.QueueUpdateDraw(func() {
		bgp.resultView.SetText(text)
	})
	bgp.app.Draw()
}
