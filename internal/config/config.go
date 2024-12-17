package config

import (
	"encoding/json"
	"os"
)

const configFileName string = ".gatorconfig.json"

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() (*Config, error) {
	var cfg Config
	url, err := getFileURL()

	if err != nil {
		return &cfg, err
	}

	file, err := os.ReadFile(url)

	if err != nil {
		return &cfg, err
	}

	err = json.Unmarshal(file, &cfg)

	if err != nil {
		return &cfg, err
	}

	return &cfg, nil
}

func getFileURL() (string, error) {

	url, err := os.UserHomeDir()

	if err != nil {
		return "", err
	}

	return url + "/" + configFileName, nil
}

func (cfg *Config) SetUser(userName string) error {

	url, err := getFileURL()

	if err != nil {
		return err
	}

	file, err := os.Create(url)

	if err != nil {
		return err
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	cfg.CurrentUserName = userName
	err = encoder.Encode(cfg)

	if err != nil {
		return err
	}

	return nil
}
