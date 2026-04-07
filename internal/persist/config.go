package persist

import "github.com/phergul/apiscope/internal/model"

// LoadConfig loads the persisted user config from disk.
func (s *Store) LoadConfig() (model.UserConfig, error) {
	path, err := s.configPath()
	if err != nil {
		return model.UserConfig{}, err
	}

	var config model.UserConfig
	if err := loadJSONFile(path, &config); err != nil {
		return model.UserConfig{}, err
	}

	return config, nil
}

// SaveConfig writes the user config file to disk.
func (s *Store) SaveConfig(config model.UserConfig) error {
	path, err := s.configPath()
	if err != nil {
		return err
	}

	return saveJSONFile(path, config)
}
