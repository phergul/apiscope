package app

import (
	"slices"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

// FindSavedEnvironment resolves one saved environment by scope and name.
func FindSavedEnvironment(environments []model.SavedEnvironment, scopeKey model.PersistenceScopeKey, name string) (model.SavedEnvironment, bool) {
	for _, environment := range environments {
		if environment.ScopeKey != scopeKey {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(environment.Name), strings.TrimSpace(name)) {
			continue
		}
		return cloneSavedEnvironment(environment), true
	}

	return model.SavedEnvironment{}, false
}

func (s *Service) loadEnvironments(scopeKey model.PersistenceScopeKey) ([]model.SavedEnvironment, error) {
	if s == nil || s.store == nil || scopeKey == "" {
		return nil, nil
	}

	environments, err := s.store.LoadEnvironments()
	if err != nil {
		s.logger.Error("load environments failed", "event", "persist_load_failed", "class", "environments", "error", err.Error())
		return nil, err
	}

	return environmentsForScope(environments, scopeKey), nil
}

// SaveEnvironment stores the current server values under one name and preserves prior bindings.
func (s *Service) SaveEnvironment(session model.SessionState, name string) ([]model.SavedEnvironment, error) {
	if s == nil || s.store == nil {
		return nil, errPersistenceUnavailable
	}

	name = strings.TrimSpace(name)
	if name == "" || session.PersistenceScopeKey == "" {
		return nil, errPersistenceUnavailable
	}

	environments, err := s.store.LoadEnvironments()
	if err != nil {
		s.logger.Error("load environments failed", "event", "persist_load_failed", "class", "environments", "error", err.Error())
		return nil, err
	}

	now := s.now().UTC()
	saved := model.SavedEnvironment{
		Name:              name,
		ScopeKey:          session.PersistenceScopeKey,
		SelectedServerURL: session.SelectedServerURL,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	for index := range environments {
		environment := environments[index]
		if environment.ScopeKey != session.PersistenceScopeKey || !strings.EqualFold(environment.Name, name) {
			continue
		}

		saved.CreatedAt = environment.CreatedAt
		saved.AuthBindings = cloneEnvironmentBindings(environment.AuthBindings)
		environments[index] = saved
		if err := s.store.SaveEnvironments(environments); err != nil {
			s.logger.Error("save environments failed", "event", "persist_save_failed", "class", "environments", "error", err.Error())
			return nil, err
		}
		return environmentsForScope(environments, session.PersistenceScopeKey), nil
	}

	environments = append(environments, saved)
	if err := s.store.SaveEnvironments(environments); err != nil {
		s.logger.Error("save environments failed", "event", "persist_save_failed", "class", "environments", "error", err.Error())
		return nil, err
	}

	return environmentsForScope(environments, session.PersistenceScopeKey), nil
}

// SaveEnvironmentBinding stores one env-var binding for a saved environment auth field.
func (s *Service) SaveEnvironmentBinding(scopeKey model.PersistenceScopeKey, environmentName, schemeName string, field model.AuthField, envVarName string) ([]model.SavedEnvironment, error) {
	if s == nil || s.store == nil || scopeKey == "" || strings.TrimSpace(environmentName) == "" || strings.TrimSpace(schemeName) == "" || field == "" {
		return nil, errPersistenceUnavailable
	}

	environments, err := s.store.LoadEnvironments()
	if err != nil {
		s.logger.Error("load environments failed", "event", "persist_load_failed", "class", "environments", "error", err.Error())
		return nil, err
	}

	envVarName = strings.TrimSpace(envVarName)
	for index := range environments {
		environment := environments[index]
		if environment.ScopeKey != scopeKey || !strings.EqualFold(environment.Name, environmentName) {
			continue
		}

		environment.AuthBindings = cloneEnvironmentBindings(environment.AuthBindings)
		updateEnvironmentBinding(&environment, schemeName, field, envVarName)
		environment.UpdatedAt = s.now().UTC()
		environments[index] = environment

		if err := s.store.SaveEnvironments(environments); err != nil {
			s.logger.Error("save environments failed", "event", "persist_save_failed", "class", "environments", "error", err.Error())
			return nil, err
		}
		return environmentsForScope(environments, scopeKey), nil
	}

	return nil, errPersistenceUnavailable
}

// DeleteEnvironment removes one saved environment without mutating the live session.
func (s *Service) DeleteEnvironment(scopeKey model.PersistenceScopeKey, name string) ([]model.SavedEnvironment, error) {
	if s == nil || s.store == nil || scopeKey == "" || strings.TrimSpace(name) == "" {
		return nil, errPersistenceUnavailable
	}

	environments, err := s.store.LoadEnvironments()
	if err != nil {
		s.logger.Error("load environments failed", "event", "persist_load_failed", "class", "environments", "error", err.Error())
		return nil, err
	}

	filtered := environments[:0]
	for _, environment := range environments {
		if environment.ScopeKey == scopeKey && strings.EqualFold(environment.Name, strings.TrimSpace(name)) {
			continue
		}
		filtered = append(filtered, environment)
	}

	if err := s.store.SaveEnvironments(filtered); err != nil {
		s.logger.Error("save environments failed", "event", "persist_save_failed", "class", "environments", "error", err.Error())
		return nil, err
	}

	return environmentsForScope(filtered, scopeKey), nil
}

func environmentsForScope(environments []model.SavedEnvironment, scopeKey model.PersistenceScopeKey) []model.SavedEnvironment {
	if len(environments) == 0 || scopeKey == "" {
		return nil
	}

	filtered := make([]model.SavedEnvironment, 0, len(environments))
	for _, environment := range environments {
		if environment.ScopeKey != scopeKey {
			continue
		}
		filtered = append(filtered, cloneSavedEnvironment(environment))
	}

	slices.SortFunc(filtered, func(left, right model.SavedEnvironment) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})

	return filtered
}
