package request

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
)

const (
	SectionEnvironment      = "Environment"
	EnvironmentSaveTarget   = "environment:save"
	EnvironmentDeleteTarget = "environment:delete"
)

func environmentRows(requirement *model.SecurityRequirement, securitySchemes map[string]model.SecurityScheme, environments []model.SavedEnvironment, appliedEnvironmentName string) []RowDescriptor {
	appliedEnvironment := currentEnvironment(environments, appliedEnvironmentName)
	rows := []RowDescriptor{{
		ID:              "environment:current",
		Kind:            RowKindEnvironmentCurrent,
		Label:           "Current environment",
		Value:           currentEnvironmentValue(appliedEnvironmentName),
		Editable:        false,
		EnvironmentName: strings.TrimSpace(appliedEnvironmentName),
	}, {
		ID:              EnvironmentSaveTarget,
		Kind:            RowKindEnvironmentSave,
		Label:           "Save current as",
		Value:           saveEnvironmentValue(appliedEnvironmentName),
		Editable:        true,
		EnvironmentName: strings.TrimSpace(appliedEnvironmentName),
	}}

	for _, binding := range app.ProjectEnvironmentBindings(requirement, securitySchemes, appliedEnvironment) {
		rows = append(rows, RowDescriptor{
			ID:              environmentBindingTarget(binding.SchemeName, binding.Field),
			Kind:            RowKindEnvironmentBinding,
			Label:           binding.Label,
			Meta:            environmentBindingMeta(binding),
			Value:           environmentBindingValue(binding.EnvVarName),
			Editable:        binding.Editable,
			AuthSchemeName:  binding.SchemeName,
			AuthField:       binding.Field,
			EnvironmentName: strings.TrimSpace(appliedEnvironmentName),
		})
	}

	for _, environment := range environments {
		rows = append(rows, RowDescriptor{
			ID:              "environment:apply:" + environment.Name,
			Kind:            RowKindEnvironmentApply,
			Label:           environment.Name,
			Value:           environmentSummary(environment),
			Meta:            "saved environment",
			Editable:        true,
			EnvironmentName: environment.Name,
		})
	}

	if strings.TrimSpace(appliedEnvironmentName) != "" {
		rows = append(rows, RowDescriptor{
			ID:              EnvironmentDeleteTarget,
			Kind:            RowKindEnvironmentDelete,
			Label:           "Delete current environment",
			Value:           strings.TrimSpace(appliedEnvironmentName),
			Meta:            "remove saved environment",
			Editable:        true,
			EnvironmentName: strings.TrimSpace(appliedEnvironmentName),
		})
	}

	return rows
}

func currentEnvironmentValue(appliedEnvironmentName string) string {
	appliedEnvironmentName = strings.TrimSpace(appliedEnvironmentName)
	if appliedEnvironmentName == "" {
		return "Session only"
	}

	return appliedEnvironmentName
}

func saveEnvironmentValue(appliedEnvironmentName string) string {
	appliedEnvironmentName = strings.TrimSpace(appliedEnvironmentName)
	if appliedEnvironmentName == "" {
		return "<new name>"
	}

	return appliedEnvironmentName
}

func environmentBindingTarget(schemeName string, field model.AuthField) string {
	return "environment:binding:" + schemeName + ":" + string(field)
}

func environmentBindingValue(envVarName string) string {
	envVarName = strings.TrimSpace(envVarName)
	if envVarName == "" {
		return "Session only"
	}

	return envVarName
}

func environmentBindingMeta(binding app.EnvironmentBindingProjection) string {
	if !binding.Editable {
		return binding.Meta + ", save environment first"
	}

	return binding.Meta + ", env var"
}

func currentEnvironment(environments []model.SavedEnvironment, appliedEnvironmentName string) *model.SavedEnvironment {
	appliedEnvironmentName = strings.TrimSpace(appliedEnvironmentName)
	if appliedEnvironmentName == "" {
		return nil
	}

	for index := range environments {
		if strings.EqualFold(strings.TrimSpace(environments[index].Name), appliedEnvironmentName) {
			return &environments[index]
		}
	}

	return nil
}

func environmentSummary(environment model.SavedEnvironment) string {
	parts := make([]string, 0, 2)
	if serverURL := strings.TrimSpace(environment.SelectedServerURL); serverURL != "" {
		parts = append(parts, serverURL)
	}

	bindingCount := countEnvironmentBindings(environment)
	switch bindingCount {
	case 0:
		parts = append(parts, "session auth only")
	case 1:
		parts = append(parts, "1 env binding")
	default:
		parts = append(parts, fmt.Sprintf("%d env bindings", bindingCount))
	}

	return strings.Join(parts, " · ")
}

func countEnvironmentBindings(environment model.SavedEnvironment) int {
	count := 0
	for _, binding := range environment.AuthBindings {
		count += len(binding.FieldEnvVars)
	}

	return count
}
