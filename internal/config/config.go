package config

import (
	"os"
	"path/filepath"
)

const (
	configDir  = ".config/gha-freeze"
	configFile = "token"
)

func GetTokenPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDir, configFile), nil
}

func SaveToken(token string) error {
	path, err := GetTokenPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(token), 0600)
}

func LoadToken() (string, error) {
	path, err := GetTokenPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(data), nil
}

func GetToken(providedToken string) string {
	if providedToken != "" {
		return providedToken
	}

	if envToken := os.Getenv("GITHUB_TOKEN"); envToken != "" {
		return envToken
	}

	if envToken := os.Getenv("GHA_FREEZE_TOKEN"); envToken != "" {
		return envToken
	}

	savedToken, _ := LoadToken()
	return savedToken
}
