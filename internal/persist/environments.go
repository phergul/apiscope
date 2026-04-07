package persist

import "github.com/phergul/apiscope/internal/model"

// LoadEnvironments loads the saved environment list from disk.
func (s *Store) LoadEnvironments() ([]model.SavedEnvironment, error) {
	path, err := s.environmentsPath()
	if err != nil {
		return nil, err
	}

	var environments []model.SavedEnvironment
	if err := loadJSONFile(path, &environments); err != nil {
		return nil, err
	}

	return environments, nil
}

// SaveEnvironments writes the saved environment list to disk.
func (s *Store) SaveEnvironments(environments []model.SavedEnvironment) error {
	path, err := s.environmentsPath()
	if err != nil {
		return err
	}

	return saveJSONFile(path, environments)
}
