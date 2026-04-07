package app

import (
	"errors"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

const (
	maxRecentSpecs               = 10
	persistenceUnavailableNotice = "Persisted data unavailable"
)

var errPersistenceUnavailable = errors.New("persistence unavailable")

// LoadConfig loads the durable user config when persistence is enabled.
func (s *Service) LoadConfig() (model.UserConfig, error) {
	if s == nil || s.store == nil {
		return model.UserConfig{}, nil
	}

	config, err := s.store.LoadConfig()
	if err != nil {
		s.logger.Error("load config failed", "event", "persist_load_failed", "class", "config", "error", err.Error())
		return model.UserConfig{}, err
	}

	return config, nil
}

// SaveThemePreference persists the selected theme when persistence is enabled.
func (s *Service) SaveThemePreference(themeName string) error {
	if s == nil || s.store == nil {
		return nil
	}

	config, err := s.store.LoadConfig()
	if err != nil {
		s.logger.Error("load config failed", "event", "persist_load_failed", "class", "config", "error", err.Error())
		return err
	}

	config.ThemeName = strings.TrimSpace(themeName)
	if err := s.store.SaveConfig(config); err != nil {
		s.logger.Error("save config failed", "event", "persist_save_failed", "class", "config", "error", err.Error())
		return err
	}

	return nil
}

func (s *Service) syncRecentSpec(rawSource string, apiSpec *model.APISpec) error {
	if s == nil || s.store == nil || apiSpec == nil {
		return nil
	}

	config, err := s.store.LoadConfig()
	if err != nil {
		s.logger.Error("load config failed", "event", "persist_load_failed", "class", "config", "error", err.Error())
		return err
	}

	config.RecentSpecs = updateRecentSpecs(config.RecentSpecs, model.RecentSpec{
		Source:       rawSource,
		Title:        apiSpec.Title,
		LastOpenedAt: s.now().UTC(),
		SourceFamily: apiSpec.SourceFamily,
	})
	if err := s.store.SaveConfig(config); err != nil {
		s.logger.Error("save config failed", "event", "persist_save_failed", "class", "config", "error", err.Error())
		return err
	}

	return nil
}

func updateRecentSpecs(existing []model.RecentSpec, recent model.RecentSpec) []model.RecentSpec {
	updated := make([]model.RecentSpec, 0, len(existing)+1)
	updated = append(updated, recent)
	for _, item := range existing {
		if strings.TrimSpace(item.Source) == strings.TrimSpace(recent.Source) {
			continue
		}
		updated = append(updated, item)
		if len(updated) == maxRecentSpecs {
			break
		}
	}

	return updated
}
