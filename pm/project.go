package pm

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type Project struct {
	Name    string    `json:"name"`
	Version string    `json:"version"`
	Url     string    `json:"url"`
	Deps    []Project `json:"deps"`
}

func LoadProject() (*Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	projConfPath := filepath.Join(cwd, "zvm.json")
	data, err := os.ReadFile(projConfPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrMissingConfig
		}

		return nil, err
	}

	var proj Project
	if err := json.Unmarshal(data, &proj); err != nil {
		return nil, err
	}

	return &proj, nil
}

func NewProject() Project {
	return Project{
		Deps: make([]Project, 0),
	}
}
