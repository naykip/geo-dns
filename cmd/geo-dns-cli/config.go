package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Server string `mapstructure:"server"`
	Token  string `mapstructure:"token"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "geo-dns-cli")
}

func configFile() string {
	return filepath.Join(configDir(), "config.yaml")
}

func initConfig() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("cannot create config dir: %w", err)
	}

	viper.SetConfigFile(configFile())
	viper.SetDefault("server", "http://localhost:8080")
	viper.SetDefault("token", "")

	if err := viper.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			// file exists but unreadable – ignore silently, defaults apply
		}
	}
	return nil
}

func saveToken(token string) error {
	viper.Set("token", token)
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	if err := viper.WriteConfigAs(configFile()); err != nil {
		return err
	}
	return os.Chmod(configFile(), 0600)
}

func currentToken() string {
	return viper.GetString("token")
}

func serverURL() string {
	return viper.GetString("server")
}
