package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Store struct {
	path string
}

func NewStore(appName string, legacyAppNames ...string) (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(configDir, appName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "state.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		for _, legacyAppName := range legacyAppNames {
			legacyPath := filepath.Join(configDir, legacyAppName, "state.json")
			data, err := os.ReadFile(legacyPath)
			if err == nil {
				_ = os.WriteFile(path, data, 0644)
				break
			}
		}
	}
	return &Store{path: path}, nil
}

func FileKey(path string, size int64, modTime int64) string {
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	return fmt.Sprintf("%s|%d|%d", filepath.Clean(path), size, modTime)
}

func (s *Store) Load() (*State, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return &State{Files: map[string]FileState{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if state.Files == nil {
		state.Files = map[string]FileState{}
	}
	return &state, nil
}

func (s *Store) Save(state *State) error {
	if state.Files == nil {
		state.Files = map[string]FileState{}
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *Store) Upsert(fileKey string, file FileState) error {
	state, err := s.Load()
	if err != nil {
		return err
	}
	file.UpdatedAt = time.Now().Unix()
	state.Files[fileKey] = file
	return s.Save(state)
}

func (s *Store) Get(fileKey string) (FileState, bool, error) {
	state, err := s.Load()
	if err != nil {
		return FileState{}, false, err
	}
	file, ok := state.Files[fileKey]
	return file, ok, nil
}
