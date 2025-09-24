package output

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
)

var tunnelColors = []string{
	ColorCyan,
	ColorYellow,
	ColorPurple,
	ColorBlue,
	ColorGreen,
}

type TunnelStatus struct {
	Name  string
	Ports map[string]string
}

type Display struct {
	mu       sync.Mutex
	statuses map[string]*TunnelStatus
	colorMap map[string]string
	colorIdx int
	printed  bool
	footer   string
}

func NewDisplay() *Display {
	return &Display{
		statuses: make(map[string]*TunnelStatus),
		colorMap: make(map[string]string),
		colorIdx: 0,
	}
}

// getColorLocked returns the color assigned to a tunnel, creating one if needed.
// Callers must hold d.mu before invoking this helper.
func (d *Display) getColorLocked(tunnelName string) string {
	if color, exists := d.colorMap[tunnelName]; exists {
		return color
	}

	color := tunnelColors[d.colorIdx%len(tunnelColors)]
	d.colorMap[tunnelName] = color
	d.colorIdx++
	return color
}

func (d *Display) UpdateStatus(tunnelName string, port string, status string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.statuses[tunnelName]; !exists {
		d.statuses[tunnelName] = &TunnelStatus{
			Name:  tunnelName,
			Ports: make(map[string]string),
		}
	}

	d.statuses[tunnelName].Ports[port] = status
	d.printStatuses()
}

func (d *Display) printStatuses() {
	// Clear screen and reprint
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%stunn is listening...%s\n", ColorGray, ColorReset)
	fmt.Println()

	names := make([]string, 0, len(d.statuses))
	for name := range d.statuses {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		status := d.statuses[name]
		color := d.getColorLocked(name)
		fmt.Printf("%s[%s]%s\n", color, name, ColorReset)

		ports := make([]string, 0, len(status.Ports))
		for port := range status.Ports {
			ports = append(ports, port)
		}
		sort.Slice(ports, func(i, j int) bool {
			leftLocal, _ := parsePort(ports[i])
			rightLocal, _ := parsePort(ports[j])

			leftNum, leftErr := strconv.Atoi(leftLocal)
			rightNum, rightErr := strconv.Atoi(rightLocal)

			if leftErr == nil && rightErr == nil {
				if leftNum == rightNum {
					return ports[i] < ports[j]
				}
				return leftNum < rightNum
			}

			if leftErr == nil {
				return true
			}
			if rightErr == nil {
				return false
			}

			if leftLocal == rightLocal {
				return ports[i] < ports[j]
			}
			return leftLocal < rightLocal
		})

		for _, port := range ports {
			portStatus := status.Ports[port]
			statusColor := ColorGray
			statusLower := strings.ToLower(portStatus)
			switch {
			case strings.HasPrefix(statusLower, "active"):
				statusColor = ColorGreen
			case strings.HasPrefix(statusLower, "error"):
				statusColor = ColorRed
			case strings.HasPrefix(statusLower, "connecting"), strings.HasPrefix(statusLower, "stopping"):
				statusColor = ColorYellow
			}

			local, remote := parsePort(port)
			fmt.Printf("    %s ➜ %s %s[%s]%s\n",
				local, remote, statusColor, portStatus, ColorReset)
		}
		fmt.Println()
	}

	footer := strings.TrimSpace(d.footer)
	if footer != "" {
		fmt.Println(footer)
	}
}

// SetFooter updates the message rendered beneath the tunnel table.
func (d *Display) SetFooter(message string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	trimmed := strings.TrimSpace(message)
	if d.footer == trimmed {
		return
	}
	d.footer = trimmed
	d.printStatuses()
}

func (d *Display) PrintError(tunnelName string, message string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	color := d.getColorLocked(tunnelName)
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		trimmed = "unknown"
	}
	fmt.Printf("\n%s[%s]%s %s[error - %s]%s\n", color, tunnelName, ColorReset, ColorRed, trimmed, ColorReset)
}

func parsePort(mapping string) (string, string) {
	mapping = strings.TrimSpace(mapping)
	if mapping == "" {
		return "", ""
	}

	parts := strings.SplitN(mapping, ":", 2)
	local := parts[0]
	remote := local
	if len(parts) == 2 && parts[1] != "" {
		remote = parts[1]
	}
	return local, remote
}
