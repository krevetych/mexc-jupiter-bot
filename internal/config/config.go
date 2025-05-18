package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"mexccrypto/types"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadConfig(path string) (*types.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	var cfg types.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	return &cfg, nil
}

func LoadSPLMap(path string) (map[string]string, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spl map: %w", err)
	}

	if len(f) == 0 {
		return nil, errors.New("empty spl.json")
	}

	m := make(map[string]string)
	if err := json.Unmarshal(f, &m); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	return m, nil
}
