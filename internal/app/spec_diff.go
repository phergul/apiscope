package app

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/phergul/apiscope/internal/model"
)

// SpecDiffCapabilityChange reports one boolean capability toggle between specs.
type SpecDiffCapabilityChange struct {
	Name string
	From bool
	To   bool
}

// SpecDiffResult reports normalized deltas between two loaded specs.
type SpecDiffResult struct {
	Changed            bool
	FromFingerprint    model.SpecFingerprint
	ToFingerprint      model.SpecFingerprint
	FromSourceFamily   model.SourceFamily
	ToSourceFamily     model.SourceFamily
	FromSourceVersion  string
	ToSourceVersion    string
	AddedOperations    []model.OperationKey
	RemovedOperations  []model.OperationKey
	ChangedOperations  []model.OperationKey
	CapabilityChanges  []SpecDiffCapabilityChange
	AddedWarnings      []model.SpecWarning
	RemovedWarnings    []model.SpecWarning
	FromOperationCount int
	ToOperationCount   int
	FromWarningCount   int
	ToWarningCount     int
}

// DiffSpecs compares two normalized specs and reports a stable, render-ready diff.
func DiffSpecs(previous, next *model.APISpec) SpecDiffResult {
	result := SpecDiffResult{}
	if previous != nil {
		result.FromFingerprint = previous.Fingerprint
		result.FromSourceFamily = previous.SourceFamily
		result.FromSourceVersion = previous.SourceVersion
		result.FromOperationCount = len(previous.Operations)
		result.FromWarningCount = len(previous.Warnings)
	}
	if next != nil {
		result.ToFingerprint = next.Fingerprint
		result.ToSourceFamily = next.SourceFamily
		result.ToSourceVersion = next.SourceVersion
		result.ToOperationCount = len(next.Operations)
		result.ToWarningCount = len(next.Warnings)
	}

	if previous == nil || next == nil {
		result.Changed = previous != next
		return result
	}

	result.AddedOperations, result.RemovedOperations, result.ChangedOperations = diffOperations(previous.Operations, next.Operations)
	result.CapabilityChanges = diffCapabilities(previous.Capabilities, next.Capabilities)
	result.AddedWarnings, result.RemovedWarnings = diffWarnings(previous.Warnings, next.Warnings)

	result.Changed = result.FromFingerprint != result.ToFingerprint ||
		result.FromSourceFamily != result.ToSourceFamily ||
		result.FromSourceVersion != result.ToSourceVersion ||
		len(result.AddedOperations) > 0 ||
		len(result.RemovedOperations) > 0 ||
		len(result.ChangedOperations) > 0 ||
		len(result.CapabilityChanges) > 0 ||
		len(result.AddedWarnings) > 0 ||
		len(result.RemovedWarnings) > 0

	return result
}

func diffOperations(previous, next []model.Operation) ([]model.OperationKey, []model.OperationKey, []model.OperationKey) {
	previousByKey := make(map[model.OperationKey]model.Operation, len(previous))
	nextByKey := make(map[model.OperationKey]model.Operation, len(next))

	for _, operation := range previous {
		previousByKey[operation.Key] = operation
	}
	for _, operation := range next {
		nextByKey[operation.Key] = operation
	}

	added := make([]model.OperationKey, 0)
	removed := make([]model.OperationKey, 0)
	changed := make([]model.OperationKey, 0)

	for key, operation := range nextByKey {
		previousOperation, ok := previousByKey[key]
		if !ok {
			added = append(added, key)
			continue
		}
		if !reflect.DeepEqual(previousOperation, operation) {
			changed = append(changed, key)
		}
	}
	for key := range previousByKey {
		if _, ok := nextByKey[key]; !ok {
			removed = append(removed, key)
		}
	}

	sortOperationKeys(added)
	sortOperationKeys(removed)
	sortOperationKeys(changed)
	return added, removed, changed
}

func sortOperationKeys(keys []model.OperationKey) {
	sort.Slice(keys, func(left, right int) bool {
		return keys[left] < keys[right]
	})
}

func diffCapabilities(previous, next model.CapabilitySet) []SpecDiffCapabilityChange {
	type capabilityField struct {
		name string
		from bool
		to   bool
	}

	fields := []capabilityField{
		{name: "SupportsSwagger2Conversion", from: previous.SupportsSwagger2Conversion, to: next.SupportsSwagger2Conversion},
		{name: "SupportsOpenAPI3", from: previous.SupportsOpenAPI3, to: next.SupportsOpenAPI3},
		{name: "SupportsCookieParameters", from: previous.SupportsCookieParameters, to: next.SupportsCookieParameters},
		{name: "SupportsRequestBodies", from: previous.SupportsRequestBodies, to: next.SupportsRequestBodies},
		{name: "SupportsServerVariables", from: previous.SupportsServerVariables, to: next.SupportsServerVariables},
		{name: "SupportsSecuritySchemes", from: previous.SupportsSecuritySchemes, to: next.SupportsSecuritySchemes},
	}

	changes := make([]SpecDiffCapabilityChange, 0, len(fields))
	for _, field := range fields {
		if field.from == field.to {
			continue
		}
		changes = append(changes, SpecDiffCapabilityChange{
			Name: field.name,
			From: field.from,
			To:   field.to,
		})
	}

	return changes
}

func diffWarnings(previous, next []model.SpecWarning) ([]model.SpecWarning, []model.SpecWarning) {
	previousMap := make(map[string]model.SpecWarning, len(previous))
	nextMap := make(map[string]model.SpecWarning, len(next))

	for _, warning := range previous {
		previousMap[warningKey(warning)] = warning
	}
	for _, warning := range next {
		nextMap[warningKey(warning)] = warning
	}

	added := make([]model.SpecWarning, 0)
	removed := make([]model.SpecWarning, 0)
	for key, warning := range nextMap {
		if _, ok := previousMap[key]; !ok {
			added = append(added, warning)
		}
	}
	for key, warning := range previousMap {
		if _, ok := nextMap[key]; !ok {
			removed = append(removed, warning)
		}
	}

	sortWarnings(added)
	sortWarnings(removed)
	return added, removed
}

func warningKey(warning model.SpecWarning) string {
	return fmt.Sprintf("%s\n%s\n%s", warning.Code, warning.Path, warning.Message)
}

func sortWarnings(warnings []model.SpecWarning) {
	sort.Slice(warnings, func(left, right int) bool {
		leftKey := warningKey(warnings[left])
		rightKey := warningKey(warnings[right])
		return leftKey < rightKey
	})
}
