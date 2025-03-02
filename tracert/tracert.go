// tracert/tracert.go
package tracert

import (
	"fmt"
	"log"
	"net"
	"os/exec"

	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Traceroute performs a traceroute to the specified IP address or hostname.
func Traceroute(ip string) (string, error) {
	cmd := exec.Command("traceroute", ip) // Use "tracert" on Windows
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// DisplayTracerouteResults displays the traceroute results in a tview TextView.
func DisplayTracerouteResults(app *tview.Application, results string) {
	resultsView := tview.NewTextView().
		SetText(results).
		SetTextAlign(tview.AlignLeft).
		SetScrollable(true)

	// Set the root to the results view
	app.SetRoot(resultsView, true)
	app.SetFocus(resultsView)
}

// ValidateIP checks if the input is a valid IP address or hostname.
func ValidateIP(ip string) bool {
	// Check if it's a valid IP address
	if net.ParseIP(ip) != nil {
		return true
	}
	// Optionally, you can add more validation for hostnames if needed
	return true // For simplicity, we assume any string is valid for hostname
}

// ///////////////////////////////
var pingStop chan struct{}

func main() {
	app := tview.NewApplication()
	pingStop = make(chan struct{})

	// Set the initial view
	showPing(app)

	// Set input capture
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case 'b':
			stopContinuousPing()
			Create(app)
		case 't': // Start traceroute
			stopContinuousPing()
			startTraceroute(app)
		case tcell.KeyEscape:
			stopContinuousPing()
			Create(app)
		}
		return event
	})

	// Handle OS interrupts for graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		stopContinuousPing()
		app.Stop()
	}()

	if err := app.Run(); err != nil {
		log.Fatalf("Error running application: %v", err)
	}
}

func showPing(app *tview.Application) {
	// Your ping view implementation
	pingView := tview.NewTextView().SetText("Ping View").SetTextAlign(tview.AlignCenter)
	app.SetRoot(pingView, true)
}

func Create(app *tview.Application) {
	// Your create view implementation
	newView := tview.NewTextView().SetText("New View").SetTextAlign(tview.AlignCenter)
	app.SetRoot(newView, true)
	app.SetFocus(newView)
}

func startTraceroute(app *tview.Application) {
	inputField := tview.NewInputField().
		SetLabel("Enter IP or Hostname: ").
		SetFieldWidth(0)

	// Create a button to trigger the traceroute
	tracertButton := tview.NewButton("Start Traceroute").SetSelectedFunc(func() {
		ip := inputField.GetText()
		if ValidateIP(ip) {
			results, err := Traceroute(ip)
			if err != nil {
				results = fmt.Sprintf("Error: %v", err)
			}
			DisplayTracerouteResults(app, results)
		} else {
			results := "Invalid IP or Hostname"
			DisplayTracerouteResults(app, results)
		}
	})

	// Create a layout for the traceroute input
	form := tview.NewForm().
		AddFormItem(inputField).
		AddButton("Start Traceroute", func() {
			tracertButton.SetSelectedFunc(tracertButton.GetSelectedFunc())
		}).
		AddButton("Cancel", func() {
			showPing(app) // Return to the ping view or main menu
		})

	// Set the root to the form for traceroute input
	app.SetRoot(form, true)
	app.SetFocus(inputField)
}

func stopContinuousPing() {
	if pingStop != nil {
		select {
		case <-pingStop:
		default:
			close(pingStop)
		}
		pingStop = make(chan struct{}) // Reinitialize to avoid panic on reuse
	}

}
