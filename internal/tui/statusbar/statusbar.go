package statusbar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Data struct {
	Source         string
	State          string
	Notice         string
	HelpHint       string
	Focus          string
	OperationLabel string
	SelectedServer string
	HasSpec        bool
	OperationCount int
	VisibleCount   int
	WarningCount   int
	FilterText     string
}

func Render(data Data, width int) string {
	parts := []string{
		fmt.Sprintf("Source: %s", data.Source),
		fmt.Sprintf("State: %s", data.State),
		fmt.Sprintf("Focus: %s", data.Focus),
	}
	if strings.TrimSpace(data.Notice) != "" {
		parts = append(parts, fmt.Sprintf("Notice: %s", data.Notice))
	}
	if strings.TrimSpace(data.OperationLabel) != "" {
		parts = append(parts, fmt.Sprintf("Operation: %s", data.OperationLabel))
	}
	if strings.TrimSpace(data.SelectedServer) != "" {
		parts = append(parts, fmt.Sprintf("Server: %s", data.SelectedServer))
	}
	if data.HasSpec {
		parts = append(parts, fmt.Sprintf("Count: %d", data.OperationCount))
		parts = append(parts, fmt.Sprintf("Visible: %d", data.VisibleCount))
		if data.WarningCount > 0 {
			parts = append(parts, fmt.Sprintf("Warnings: %d", data.WarningCount))
		}
	}
	if strings.TrimSpace(data.FilterText) != "" {
		parts = append(parts, fmt.Sprintf("Filter: %s", data.FilterText))
	}
	parts = append(parts, "Keys: 1-4 switch Tab cycle z zoom q quit")

	left := strings.Join(parts, " | ")
	right := strings.TrimSpace(data.HelpHint)
	if right == "" || width <= 0 {
		return left
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	if leftWidth+rightWidth+1 > width {
		return left + " " + right
	}

	return left + strings.Repeat(" ", width-leftWidth-rightWidth) + right
}
