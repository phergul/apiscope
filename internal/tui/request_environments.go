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
	if strings.TrimSpace(row.AuthSchemeName) == "" || row.AuthField == "" {
		return false
	}

	if m.requestUI.authSourceOverrides == nil {
		m.requestUI.authSourceOverrides = make(map[string]requestui.AuthSourceOverride)
	}

	key := row.AuthSchemeName + ":" + string(row.AuthField)
	source = strings.TrimSpace(source)
	if source == "" {
		m.requestUI.authSourceOverrides[key] = requestui.AuthSourceOverride{UseSession: true}
		if scheme, ok := m.securitySchemes()[row.AuthSchemeName]; ok && strings.TrimSpace(scheme.Name) != "" {
			m.requestUI.authSourceOverrides[scheme.Name+":"+string(row.AuthField)] = requestui.AuthSourceOverride{UseSession: true}
		}
		draft := m.ensureSelectedRequestDraft()
		if draft != nil && draft.BodyPartEncoding != nil {
			delete(draft.BodyPartEncoding, "auth:env:"+row.AuthSchemeName+":"+string(row.AuthField))
			if scheme, ok := m.securitySchemes()[row.AuthSchemeName]; ok && strings.TrimSpace(scheme.Name) != "" {
				delete(draft.BodyPartEncoding, "auth:env:"+scheme.Name+":"+string(row.AuthField))
			}
		}
		m.viewState.Notice = "Using session value: " + row.Label
		return true
	}

	m.requestUI.authSourceOverrides[key] = requestui.AuthSourceOverride{EnvVarName: source}
	if scheme, ok := m.securitySchemes()[row.AuthSchemeName]; ok && strings.TrimSpace(scheme.Name) != "" {
		m.requestUI.authSourceOverrides[scheme.Name+":"+string(row.AuthField)] = requestui.AuthSourceOverride{EnvVarName: source}
	}
	if draft := m.ensureSelectedRequestDraft(); draft != nil {
		if draft.BodyPartEncoding == nil {
			draft.BodyPartEncoding = make(map[string]string)
		}
		draft.BodyPartEncoding["auth:env:"+row.AuthSchemeName+":"+string(row.AuthField)] = source
		if scheme, ok := m.securitySchemes()[row.AuthSchemeName]; ok && strings.TrimSpace(scheme.Name) != "" {
			draft.BodyPartEncoding["auth:env:"+scheme.Name+":"+string(row.AuthField)] = source
		}
	}
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

func (m *Model) saveAuthField(row requestui.RowDescriptor, buffer string) bool {
	if row.Kind != requestui.RowKindAuthField {
		return false
	}

	mode := m.requestUI.authEditSourceMode
	if mode == "" {
		mode = requestui.AuthSourceModeSession
	}
	if mode == requestui.AuthSourceModeEnv {
		return m.saveAuthSource(row, buffer)
	}

	if m.requestUI.authSourceOverrides == nil {
		m.requestUI.authSourceOverrides = make(map[string]requestui.AuthSourceOverride)
	}
	key := row.AuthSchemeName + ":" + string(row.AuthField)
	m.requestUI.authSourceOverrides[key] = requestui.AuthSourceOverride{UseSession: true}

	scheme, ok := m.securitySchemes()[row.AuthSchemeName]
	if !ok {
		m.viewState.Notice = "Auth scheme not found"
		return false
	}
	app.SetAuthField(&m.session, scheme, row.AuthField, buffer)
	m.viewState.Notice = "Using session value: " + row.Label
	return true
}

func (m *Model) saveBodyPartEncoding(row requestui.RowDescriptor, contentType string) bool {
	if row.Kind != requestui.RowKindBodyPartEncoding {
		return false
	}

	label := strings.TrimSuffix(strings.TrimSpace(row.Label), " content type")
	if label == "" {
		return false
	}

	if app.SetDraftBodyPartContentType(&m.session, m.resolvedSelectedOperation(), label, contentType) == nil {
		return false
	}
	if strings.TrimSpace(contentType) == "" {
		m.viewState.Notice = "Cleared part content type: " + label
	} else {
		m.viewState.Notice = "Saved part content type: " + label
	}
	return true
}

func (m *Model) toggleAuthFieldSourceMode() bool {
	row, ok := activeRequestRow(m.activeRequestRows(), m.viewState.RequestActiveRow)
	if !ok || row.Kind != requestui.RowKindAuthField || m.viewState.RequestEditKind != model.RequestEditKindField {
		return false
	}

	if m.requestUI.authEditSourceMode == requestui.AuthSourceModeEnv {
		m.requestUI.authEditSourceMode = requestui.AuthSourceModeSession
		scheme, ok := m.securitySchemes()[row.AuthSchemeName]
		value := ""
		if ok {
			value = app.AuthFieldValue(m.session.AuthState[scheme.Name], row.AuthField)
		}
		m.widgets.requestFieldInput.SetValue(value)
		m.viewState.RequestEditBuffer = value
		return true
	}

	m.requestUI.authEditSourceMode = requestui.AuthSourceModeEnv
	m.widgets.requestFieldInput.SetValue(strings.TrimSpace(row.AuthEnvVarName))
	m.viewState.RequestEditBuffer = strings.TrimSpace(row.AuthEnvVarName)
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

func isEnvironmentBindingTarget(target string) bool {
	return strings.HasPrefix(target, "environment:binding:")
}

func isBodyEncodingTarget(target string) bool {
	return strings.HasPrefix(target, app.ValidationTargetBodyEncodingPrefix)
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
