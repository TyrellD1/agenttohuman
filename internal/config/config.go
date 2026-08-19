package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const Version = 1

type MethodConfig map[string]any

func (m MethodConfig) String(key string) string {
	if m == nil {
		return ""
	}
	value, _ := m[key].(string)
	return value
}

func (m MethodConfig) SetString(key, value string) MethodConfig {
	next := MethodConfig{}
	for k, v := range m {
		next[k] = v
	}
	if value == "" {
		delete(next, key)
	} else {
		next[key] = value
	}
	return next
}

type AppConfig struct {
	Version       int                     `json:"version"`
	DefaultMethod *string                 `json:"defaultMethod"`
	Methods       map[string]MethodConfig `json:"methods"`
}

func Empty() AppConfig {
	return AppConfig{
		Version: Version,
		Methods: map[string]MethodConfig{},
	}
}

func Dir() string {
	if home := os.Getenv("AGENTTOHUMAN_HOME"); home != "" {
		return home
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".agenttohuman")
	}
	return filepath.Join(userHome, ".agenttohuman")
}

func Path() string {
	return filepath.Join(Dir(), "config.json")
}

func Load() (AppConfig, error) {
	path := Path()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Empty(), nil
		}
		return AppConfig{}, err
	}
	var parsed AppConfig
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return AppConfig{}, fmt.Errorf("config at %s is not valid JSON", path)
	}
	if parsed.Methods == nil {
		parsed.Methods = map[string]MethodConfig{}
	}
	if parsed.Version == 0 {
		parsed.Version = Version
	}
	return parsed, nil
}

func Save(cfg AppConfig) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if cfg.Methods == nil {
		cfg.Methods = map[string]MethodConfig{}
	}
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(body, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func Settings(cfg AppConfig, methodID string) MethodConfig {
	if cfg.Methods == nil {
		return MethodConfig{}
	}
	if settings, ok := cfg.Methods[methodID]; ok && settings != nil {
		return settings
	}
	return MethodConfig{}
}

func WithSettings(cfg AppConfig, methodID string, settings MethodConfig) AppConfig {
	next := cfg
	next.Methods = map[string]MethodConfig{}
	for id, existing := range cfg.Methods {
		next.Methods[id] = existing
	}
	next.Methods[methodID] = settings
	if next.DefaultMethod == nil || *next.DefaultMethod == "" {
		id := methodID
		next.DefaultMethod = &id
	}
	return next
}

func DefaultID(cfg AppConfig) string {
	if cfg.DefaultMethod == nil {
		return ""
	}
	return *cfg.DefaultMethod
}
