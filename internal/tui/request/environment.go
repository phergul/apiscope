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
	EnvironmentUnloadTarget = "environment:unload"
	EnvironmentDeleteTarget = "environment:delete"
)

func environmentRows(requirement *model.SecurityRequirement, securitySchemes map[string]model.SecurityScheme, environments []model.SavedEnvironment, appliedEnvironmentName string) []RowDescriptor {
	appliedEnvironment := currentEnvironment(environments, appliedEnvironmentName)
	rows := []RowDescriptor{{
		ID:              "environment:current",
		Kind:            RowKindEnvironmentCurrent,
		Label:           "Loaded environment",
		Meta:            "session state",
		Value:           currentEnvironmentValue(appliedEnvironmentName),
		Editable:        false,
		EnvironmentName: strings.TrimSpace(appliedEnvironmentName),
	}}

	if strings.TrimSpace(appliedEnvironmentName) != "" {
		rows = append(rows, RowDescriptor{
			ID:              EnvironmentUnloadTarget,
			Kind:            RowKindEnvironmentUnload,
			Label:           "Unload environment",
			Meta:            "keep current session values",
			Value:           strings.TrimSpace(appliedEnvironmentName),
			Editable:        true,
			EnvironmentName: strings.TrimSpace(appliedEnvironmentName),
		})
	}

	rows = append(rows, RowDescriptor{
		ID:              EnvironmentSaveTarget,
		Kind:            RowKindEnvironmentSave,
		Label:           "Save session as",
		Meta:            "Enter saves or updates",
		Value:           saveEnvironmentValue(appliedEnvironmentName),
		Editable:        true,
		EnvironmentName: strings.TrimSpace(appliedEnvironmentName),
	})

	for _, environment := range environments {
		meta := "saved environment, Enter loads"
		if strings.EqualFold(strings.TrimSpace(environment.Name), strings.TrimSpace(appliedEnvironmentName)) {
			meta = "loaded now, Enter reloads"
		}

		rows = append(rows, RowDescriptor{
			ID:              "environment:apply:" + environment.Name,
			Kind:            RowKindEnvironmentApply,
			Label:           environment.Name,
			Value:           environmentSummary(environment),
			Meta:            meta,
			Editable:        true,
			EnvironmentName: environment.Name,
		})
	}

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

	if strings.TrimSpace(appliedEnvironmentName) != "" {
		rows = append(rows, RowDescriptor{
			ID:              EnvironmentDeleteTarget,
			Kind:            RowKindEnvironmentDelete,
			Label:           "Delete saved environment",
			Value:           strings.TrimSpace(appliedEnvironmentName),
			Meta:            "Enter confirms delete",
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

func authSourceTarget(schemeName string, field model.AuthField) string {
	return "auth:source:" + schemeName + ":" + string(field)
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
		return binding.Meta + ", save an environment to enable binding"
	}

	return binding.Meta + ", env var binding"
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
		parts = append(parts, "server: "+serverURL)
	}

	bindingCount := countEnvironmentBindings(environment)
	switch bindingCount {
	case 0:
		parts = append(parts, "auth: session only")
	case 1:
		parts = append(parts, "auth: 1 env binding")
	default:
		parts = append(parts, fmt.Sprintf("auth: %d env bindings", bindingCount))
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
