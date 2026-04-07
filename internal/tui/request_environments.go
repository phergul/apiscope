package tui

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	requestui "github.com/phergul/apiscope/internal/tui/request"
)

func (m *Model) applyEnvironmentByName(name string) {
	environment, ok := app.FindSavedEnvironment(m.persisted.environments, m.session.PersistenceScopeKey, name)
	if !ok {
		m.viewState.Notice = "Environment not found"
		return
	}

	result := m.service.ApplyEnvironment(&m.session, environment)
	m.requestUI.appliedEnvironmentName = environment.Name
	m.clearRequestValidation()
	m.viewState.Notice = environmentApplyNotice("Environment applied", result.MissingEnvVars)
}

func (m *Model) saveCurrentEnvironment(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		m.viewState.Notice = "Environment name required"
		return false
	}

	environments, err := m.service.SaveEnvironment(m.session, name)
	if err != nil {
		m.viewState.Notice = "Environment not saved"
		return false
	}

	m.persisted.environments = environments
	m.requestUI.appliedEnvironmentName = name
	m.viewState.Notice = "Environment saved"
	return true
}

func (m *Model) saveEnvironmentBinding(row requestui.RowDescriptor, envVarName string) bool {
	environmentName := strings.TrimSpace(row.EnvironmentName)
	if environmentName == "" {
		m.viewState.Notice = "Save environment first"
		return false
	}

	environments, err := m.service.SaveEnvironmentBinding(m.session.PersistenceScopeKey, environmentName, row.AuthSchemeName, model.AuthField(row.AuthField), envVarName)
	if err != nil {
		m.viewState.Notice = "Environment binding not saved"
		return false
	}

	m.persisted.environments = environments
	environment, ok := app.FindSavedEnvironment(environments, m.session.PersistenceScopeKey, environmentName)
	if !ok {
		m.viewState.Notice = "Environment binding not saved"
		return false
	}

	result := m.service.ApplyEnvironment(&m.session, environment)
	m.requestUI.appliedEnvironmentName = environment.Name
	m.clearRequestValidation()
	m.viewState.Notice = environmentApplyNotice("Environment binding saved", result.MissingEnvVars)
	return true
}

func (m *Model) deleteCurrentEnvironment() bool {
	name := strings.TrimSpace(m.viewState.RequestEditBuffer)
	if name == "" {
		return false
	}

	environments, err := m.service.DeleteEnvironment(m.session.PersistenceScopeKey, name)
	if err != nil {
		m.viewState.Notice = "Environment not deleted"
		return false
	}

	m.persisted.environments = environments
	m.requestUI.appliedEnvironmentName = ""
	m.viewState.Notice = "Environment deleted"
	return true
}

func (m *Model) syncAppliedEnvironmentMarker() {
	name := strings.TrimSpace(m.requestUI.appliedEnvironmentName)
	if name == "" {
		return
	}

	environment, ok := app.FindSavedEnvironment(m.persisted.environments, m.session.PersistenceScopeKey, name)
	if !ok || !m.service.EnvironmentMatchesSession(m.session, environment) {
		m.requestUI.appliedEnvironmentName = ""
	}
}

func isEnvironmentSaveTarget(target string) bool {
	return target == requestui.EnvironmentSaveTarget
}

func isEnvironmentDeleteTarget(target string) bool {
	return target == requestui.EnvironmentDeleteTarget
}

func isEnvironmentBindingTarget(target string) bool {
	return strings.HasPrefix(target, "environment:binding:")
}

func shouldSyncAppliedEnvironment(row requestui.RowDescriptor) bool {
	return row.Kind == requestui.RowKindServer || row.Kind == requestui.RowKindAuthField
}

func activeRequestRow(rows []requestui.RowDescriptor, activeRow int) (requestui.RowDescriptor, bool) {
	if len(rows) == 0 || activeRow < 0 {
		return requestui.RowDescriptor{}, false
	}

	index := requestui.ClampActiveRow(activeRow, len(rows))
	return rows[index], true
}

func environmentApplyNotice(prefix string, missingEnvVars []string) string {
	switch len(missingEnvVars) {
	case 0:
		return prefix
	case 1:
		return fmt.Sprintf("%s; missing %s", prefix, missingEnvVars[0])
	default:
		return fmt.Sprintf("%s; missing %d env vars", prefix, len(missingEnvVars))
	}
}
