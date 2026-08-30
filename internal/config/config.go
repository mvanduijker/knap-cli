// Package config stores the API URL and default account in a plain JSON file,
// and the token in the OS keyring with a 0600 file as fallback.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "knap"
	keyringUser    = "api-token"
)

type Config struct {
	APIURL  string `json:"api_url,omitempty"`
	Account string `json:"account,omitempty"`
}

// Dir is ~/.config/knap, or $KNAP_CONFIG_DIR when set (tests, XDG setups).
func Dir() (string, error) {
	if dir := os.Getenv("KNAP_CONFIG_DIR"); dir != "" {
		return dir, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(base, "knap"), nil
}

func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(raw, &config)

	return config, err
}

func Save(config Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

// Token prefers KNAP_API_TOKEN, then the keyring, then the fallback file.
func Token() (string, error) {
	if token := os.Getenv("KNAP_API_TOKEN"); token != "" {
		return token, nil
	}

	if token, err := keyring.Get(keyringService, keyringUser); err == nil {
		return token, nil
	}

	path, err := credentialsPath()
	if err != nil {
		return "", err
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	var credentials struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(raw, &credentials)

	return credentials.Token, err
}

func SaveToken(token string) error {
	if err := keyring.Set(keyringService, keyringUser, token); err == nil {
		return nil
	}

	path, err := credentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	raw, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

// DeleteToken clears both stores, so a keyring hit never leaves a stale file
// behind (or the other way around).
func DeleteToken() error {
	if err := keyring.Delete(keyringService, keyringUser); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}

	path, err := credentialsPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

func configPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.json"), nil
}

func credentialsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "credentials.json"), nil
}
