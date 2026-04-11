package specdiff

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
)

// Render builds a readable popup body for one spec diff snapshot.
func Render(diff app.SpecDiffResult) string {
	lines := []string{
		fmt.Sprintf("Operations: %d -> %d", diff.FromOperationCount, diff.ToOperationCount),
		fmt.Sprintf("- added: %d", len(diff.AddedOperations)),
		fmt.Sprintf("- removed: %d", len(diff.RemovedOperations)),
		fmt.Sprintf("- changed: %d", len(diff.ChangedOperations)),
	}

	if len(diff.AddedOperations) > 0 {
		lines = append(lines, sectionLines("Added operations", operationLines(diff.AddedOperations, 8))...)
	}
	if len(diff.RemovedOperations) > 0 {
		lines = append(lines, sectionLines("Removed operations", operationLines(diff.RemovedOperations, 8))...)
	}
	if len(diff.ChangedOperations) > 0 {
		lines = append(lines, sectionLines("Changed operations", operationLines(diff.ChangedOperations, 8))...)
	}

	if len(diff.CapabilityChanges) > 0 {
		capabilityLines := make([]string, 0, len(diff.CapabilityChanges))
		for _, change := range diff.CapabilityChanges {
			capabilityLines = append(capabilityLines, fmt.Sprintf("- %s: %t -> %t", change.Name, change.From, change.To))
		}
		lines = append(lines, sectionLines("Capability changes", capabilityLines)...)
	}

	if len(diff.AddedWarnings) > 0 || len(diff.RemovedWarnings) > 0 {
		lines = append(lines, "", fmt.Sprintf("Warnings: %d -> %d", diff.FromWarningCount, diff.ToWarningCount))
		if len(diff.AddedWarnings) > 0 {
			lines = append(lines, warningSectionLines("Added warnings", diff.AddedWarnings, 4)...)
		}
		if len(diff.RemovedWarnings) > 0 {
			lines = append(lines, warningSectionLines("Removed warnings", diff.RemovedWarnings, 4)...)
		}
	}

	if !diff.Changed {
		lines = append(lines, "", "No normalized changes detected.")
	}

	return strings.Join(lines, "\n")
}

func sectionLines(title string, rows []string) []string {
	lines := []string{"", title + ":"}
	lines = append(lines, rows...)
	return lines
}

func operationLines(keys []model.OperationKey, maxRows int) []string {
	lines := make([]string, 0, min(len(keys), maxRows)+1)
	for index, key := range keys {
		if index == maxRows {
			lines = append(lines, fmt.Sprintf("- ... +%d more", len(keys)-maxRows))
			return lines
		}
		lines = append(lines, "- "+key.String())
	}

	return lines
}

func warningSectionLines(title string, warnings []model.SpecWarning, maxRows int) []string {
	lines := []string{"", title + ":"}
	for index, warning := range warnings {
		if index == maxRows {
			lines = append(lines, fmt.Sprintf("- ... +%d more", len(warnings)-maxRows))
			return lines
		}
		path := strings.TrimSpace(warning.Path)
		if path == "" {
			path = "(no path)"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", warning.Code, path))
	}

	return lines
}

func fallbackFingerprint(value model.SpecFingerprint) string {
	if strings.TrimSpace(string(value)) == "" {
		return "(none)"
	}
	return string(value)
}

func fallbackFamily(value model.SourceFamily) string {
	if strings.TrimSpace(string(value)) == "" {
		return "unknown"
	}
	return string(value)
}

func fallbackVersion(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(unknown)"
	}
	return value
}
