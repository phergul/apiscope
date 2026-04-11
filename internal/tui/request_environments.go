package tui

import (
	"fmt"
	"os"
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

	m.requestUI.authSourceOverrides = nil
	result := m.service.ApplyEnvironment(&m.session, environment)
	m.requestUI.appliedEnvironmentName = environment.Name
	m.clearRequestValidation()
	m.focusEnvironmentRow("environment:apply:" + environment.Name)
	m.viewState.Notice = environmentApplyNotice("Loaded environment: "+environment.Name, result.MissingEnvVars)
}

func (m *Model) unloadEnvironment() {
	name := strings.TrimSpace(m.requestUI.appliedEnvironmentName)
	if name == "" {
		m.viewState.Notice = "No environment loaded"
		return
	}

	m.requestUI.appliedEnvironmentName = ""
	m.requestUI.authSourceOverrides = nil
	m.focusEnvironmentRow(requestui.EnvironmentSaveTarget)
	m.viewState.Notice = "Unloaded environment: " + name
}

func (m *Model) saveCurrentEnvironment(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		m.viewState.Notice = "Enter an environment name"
		return false
	}

	environments, err := m.service.SaveEnvironment(m.session, name)
	if err != nil {
		m.viewState.Notice = "Could not save environment"
		return false
	}

	m.persisted.environments = environments
	m.requestUI.appliedEnvironmentName = name
	m.requestUI.authSourceOverrides = nil
	m.focusEnvironmentRow("environment:apply:" + name)
	m.viewState.Notice = "Saved environment: " + name
	return true
}

func (m *Model) saveEnvironmentBinding(row requestui.RowDescriptor, envVarName string) bool {
	environmentName := strings.TrimSpace(row.EnvironmentName)
	if environmentName == "" {
		m.viewState.Notice = "Save an environment before adding bindings"
		return false
	}

	environments, err := m.service.SaveEnvironmentBinding(m.session.PersistenceScopeKey, environmentName, row.AuthSchemeName, model.AuthField(row.AuthField), envVarName)
	if err != nil {
		m.viewState.Notice = "Could not save environment binding"
		return false
	}

	m.persisted.environments = environments
	environment, ok := app.FindSavedEnvironment(environments, m.session.PersistenceScopeKey, environmentName)
	if !ok {
		m.viewState.Notice = "Could not save environment binding"
		return false
	}

	result := m.service.ApplyEnvironment(&m.session, environment)
	m.requestUI.appliedEnvironmentName = environment.Name
	m.clearRequestValidation()
	m.focusEnvironmentRow(row.ID)
	m.viewState.Notice = environmentApplyNotice("Saved binding: "+row.Label, result.MissingEnvVars)
	return true
}

func (m *Model) saveAuthSource(row requestui.RowDescriptor, source string) bool {
	if row.Kind != requestui.RowKindAuthSource || strings.TrimSpace(row.AuthSchemeName) == "" || row.AuthField == "" {
		return false
	}

	if m.requestUI.authSourceOverrides == nil {
		m.requestUI.authSourceOverrides = make(map[string]requestui.AuthSourceOverride)
	}

	key := row.AuthSchemeName + ":" + string(row.AuthField)
	source = strings.TrimSpace(source)
	if source == "" {
		m.requestUI.authSourceOverrides[key] = requestui.AuthSourceOverride{UseSession: true}
		m.viewState.Notice = "Using session value: " + row.Label
		return true
	}

	m.requestUI.authSourceOverrides[key] = requestui.AuthSourceOverride{EnvVarName: source}
	value, ok := os.LookupEnv(source)
	scheme, schemeOK := m.securitySchemes()[row.AuthSchemeName]
	if !ok || strings.TrimSpace(value) == "" {
		if schemeOK {
			app.SetAuthField(&m.session, scheme, row.AuthField, "")
		}
		m.viewState.Notice = "Using env var source: " + source + " (currently missing)"
		return true
	}

	if !schemeOK {
		m.viewState.Notice = "Auth scheme not found"
		return false
	}
	app.SetAuthField(&m.session, scheme, row.AuthField, value)
	m.viewState.Notice = "Using env var source: " + source
	return true
}

func (m *Model) deleteCurrentEnvironment() bool {
	name := strings.TrimSpace(m.viewState.RequestEditBuffer)
	if name == "" {
		return false
	}

	environments, err := m.service.DeleteEnvironment(m.session.PersistenceScopeKey, name)
	if err != nil {
		m.viewState.Notice = "Could not delete environment"
		return false
	}

	m.persisted.environments = environments
	m.requestUI.appliedEnvironmentName = ""
	m.requestUI.authSourceOverrides = nil
	m.focusEnvironmentRow(requestui.EnvironmentSaveTarget)
	m.viewState.Notice = "Deleted environment: " + name
	return true
}

func (m *Model) focusEnvironmentRow(target string) {
	if strings.TrimSpace(target) == "" {
		return
	}

	rows := m.activeRequestRows()
	if index := requestui.RowIndexByID(rows, target); index >= 0 {
		m.viewState.RequestActiveRow = index
	}
	m.syncActiveRequestRow()
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

func isAuthSourceTarget(target string) bool {
	return strings.HasPrefix(target, "auth:source:")
}

func isEnvironmentBindingTarget(target string) bool {
	return strings.HasPrefix(target, "environment:binding:")
}

func shouldSyncAppliedEnvironment(row requestui.RowDescriptor) bool {
	return row.Kind == requestui.RowKindServer
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
