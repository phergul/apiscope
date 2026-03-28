package statusbar

import (
	"fmt"
	"strings"
)

type Data struct {
	Source         string
	State          string
	Focus          string
	OperationLabel string
	HasSpec        bool
	OperationCount int
	VisibleCount   int
	WarningCount   int
	FilterText     string
}

func Render(data Data) string {
	parts := []string{
		fmt.Sprintf("Source: %s", data.Source),
		fmt.Sprintf("State: %s", data.State),
		fmt.Sprintf("Focus: %s", data.Focus),
	}
	if strings.TrimSpace(data.OperationLabel) != "" {
		parts = append(parts, fmt.Sprintf("Operation: %s", data.OperationLabel))
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

	return strings.Join(parts, " | ")
}
