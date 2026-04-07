package app

import (
	"os"
	"slices"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

var environmentBindingFieldOrder = []model.AuthField{
	model.AuthFieldAPIKey,
	model.AuthFieldBearerToken,
	model.AuthFieldUsername,
	model.AuthFieldPassword,
}

// EnvironmentBindingProjection describes one editable env-var binding row.
type EnvironmentBindingProjection struct {
	SchemeName string
	Field      model.AuthField
	Label      string
	Meta       string
	EnvVarName string
	Editable   bool
}

// ApplyEnvironmentResult reports the live-session changes made while applying one environment.
type ApplyEnvironmentResult struct {
	Changed        bool
	MissingEnvVars []string
}

// ApplyEnvironment resolves one saved environment into the live session state.
func (s *Service) ApplyEnvironment(session *model.SessionState, environment model.SavedEnvironment) ApplyEnvironmentResult {
	if session == nil {
		return ApplyEnvironmentResult{}
	}

	changed := false
	if strings.TrimSpace(environment.SelectedServerURL) != "" {
		changed = SetSelectedServer(session, environment.SelectedServerURL) || changed
	}

	resolvedAuth, missingEnvVars := s.resolveEnvironmentAuth(environment, sessionSecuritySchemes(*session))
	if !authStateEqual(session.AuthState, resolvedAuth) {
		session.AuthState = cloneAuthState(resolvedAuth)
		changed = true
	}

	return ApplyEnvironmentResult{
		Changed:        changed,
		MissingEnvVars: missingEnvVars,
	}
}

// EnvironmentMatchesSession reports whether the saved environment still matches session state.
func (s *Service) EnvironmentMatchesSession(session model.SessionState, environment model.SavedEnvironment) bool {
	resolvedAuth, _ := s.resolveEnvironmentAuth(environment, sessionSecuritySchemes(session))
	return strings.TrimSpace(session.SelectedServerURL) == strings.TrimSpace(environment.SelectedServerURL) &&
		authStateEqual(session.AuthState, resolvedAuth)
}

// ProjectEnvironmentBindings projects the supported env-var bindings for one environment.
func ProjectEnvironmentBindings(requirement *model.SecurityRequirement, securitySchemes map[string]model.SecurityScheme, environment *model.SavedEnvironment) []EnvironmentBindingProjection {
	if requirement == nil || len(requirement.Alternatives) == 0 || len(securitySchemes) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	rows := make([]EnvironmentBindingProjection, 0)
	for _, alternative := range requirement.Alternatives {
		for _, ref := range alternative.Schemes {
			scheme, ok := securitySchemes[ref.Name]
			if !ok {
				continue
			}

			fields := SupportedAuthFields(scheme)
			for _, field := range fields {
				rowKey := scheme.Name + ":" + string(field)
				if _, ok := seen[rowKey]; ok {
					continue
				}
				seen[rowKey] = struct{}{}

				rows = append(rows, EnvironmentBindingProjection{
					SchemeName: scheme.Name,
					Field:      field,
					Label:      AuthFieldLabel(scheme, field),
					Meta:       AuthFieldMeta(scheme, field),
					EnvVarName: environmentBindingEnvVar(environment, scheme.Name, field),
					Editable:   environment != nil && strings.TrimSpace(environment.Name) != "",
				})
			}
		}
	}

	return rows
}

func sessionSecuritySchemes(session model.SessionState) map[string]model.SecurityScheme {
	if session.Spec == nil {
		return nil
	}

	return session.Spec.SecuritySchemes
}

func (s *Service) resolveEnvironmentAuth(environment model.SavedEnvironment, securitySchemes map[string]model.SecurityScheme) (map[string]model.AuthValue, []string) {
	return resolveEnvironmentAuth(environment, securitySchemes, s.lookupEnvFunc())
}

func (s *Service) lookupEnvFunc() func(string) (string, bool) {
	if s != nil && s.lookupEnv != nil {
		return s.lookupEnv
	}

	return os.LookupEnv
}

func resolveEnvironmentAuth(environment model.SavedEnvironment, securitySchemes map[string]model.SecurityScheme, lookupEnv func(string) (string, bool)) (map[string]model.AuthValue, []string) {
	if len(environment.AuthBindings) == 0 || len(securitySchemes) == 0 {
		return nil, nil
	}
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	resolved := make(map[string]model.AuthValue)
	missingSet := make(map[string]struct{})
	schemeNames := make([]string, 0, len(environment.AuthBindings))
	for schemeName := range environment.AuthBindings {
		schemeNames = append(schemeNames, schemeName)
	}
	slices.Sort(schemeNames)

	for _, schemeName := range schemeNames {
		scheme, ok := securitySchemes[schemeName]
		if !ok {
			continue
		}

		binding := environment.AuthBindings[schemeName]
		for _, field := range environmentBindingFieldOrder {
			envVarName := strings.TrimSpace(binding.FieldEnvVars[field])
			if envVarName == "" {
				continue
			}
			value, ok := lookupEnv(envVarName)
			if !ok || strings.TrimSpace(value) == "" {
				missingSet[envVarName] = struct{}{}
				continue
			}

			setAuthFieldValue(resolved, scheme, field, value)
		}
	}

	if len(resolved) == 0 {
		resolved = nil
	}
	if len(missingSet) == 0 {
		return resolved, nil
	}

	missing := make([]string, 0, len(missingSet))
	for envVarName := range missingSet {
		missing = append(missing, envVarName)
	}
	slices.Sort(missing)

	return resolved, missing
}

func updateEnvironmentBinding(environment *model.SavedEnvironment, schemeName string, field model.AuthField, envVarName string) {
	if environment == nil {
		return
	}
	if environment.AuthBindings == nil {
		environment.AuthBindings = make(map[string]model.SavedAuthBinding)
	}

	binding := environment.AuthBindings[schemeName]
	if binding.FieldEnvVars == nil {
		binding.FieldEnvVars = make(map[model.AuthField]string)
	}

	if envVarName == "" {
		delete(binding.FieldEnvVars, field)
		if len(binding.FieldEnvVars) == 0 {
			delete(environment.AuthBindings, schemeName)
		} else {
			environment.AuthBindings[schemeName] = binding
		}
		if len(environment.AuthBindings) == 0 {
			environment.AuthBindings = nil
		}
		return
	}

	binding.FieldEnvVars[field] = envVarName
	environment.AuthBindings[schemeName] = binding
}

func environmentBindingEnvVar(environment *model.SavedEnvironment, schemeName string, field model.AuthField) string {
	if environment == nil || len(environment.AuthBindings) == 0 {
		return ""
	}

	binding, ok := environment.AuthBindings[schemeName]
	if !ok {
		return ""
	}

	return strings.TrimSpace(binding.FieldEnvVars[field])
}

func cloneSavedEnvironment(environment model.SavedEnvironment) model.SavedEnvironment {
	environment.AuthBindings = cloneEnvironmentBindings(environment.AuthBindings)
	return environment
}

func cloneEnvironmentBindings(bindings map[string]model.SavedAuthBinding) map[string]model.SavedAuthBinding {
	if len(bindings) == 0 {
		return nil
	}

	cloned := make(map[string]model.SavedAuthBinding, len(bindings))
	for schemeName, binding := range bindings {
		cloned[schemeName] = model.SavedAuthBinding{
			FieldEnvVars: cloneBindingFieldMap(binding.FieldEnvVars),
		}
	}

	return cloned
}

func cloneBindingFieldMap(fields map[model.AuthField]string) map[model.AuthField]string {
	if len(fields) == 0 {
		return nil
	}

	cloned := make(map[model.AuthField]string, len(fields))
	for field, envVarName := range fields {
		cloned[field] = envVarName
	}

	return cloned
}

func authStateEqual(left, right map[string]model.AuthValue) bool {
	if len(left) != len(right) {
		return false
	}

	for name, leftValue := range left {
		if right[name] != leftValue {
			return false
		}
	}

	return true
}
