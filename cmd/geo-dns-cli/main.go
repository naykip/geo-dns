package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "geo-dns-cli",
	Short: "CLI client for geo-dns server administration",
	Long: `geo-dns-cli — CLI-клиент для управления geo-dns сервером.

Конфиг хранится в ~/.config/geo-dns-cli/config.yaml (права 0600).
Флаги --server и --token перекрывают значения из конфига.

Быстрый старт:
  geo-dns-cli login --server http://localhost:8080 -p mypassword
  geo-dns-cli zones list
  geo-dns-cli zones add --file zone.json

Примеры подключения:
  # Локальный сервер
  geo-dns-cli --server http://localhost:8080 login -p secret

  # Удалённый сервер по HTTPS
  geo-dns-cli --server https://dns.example.com login -p secret

  # Использовать токен напрямую (без login)
  geo-dns-cli --server https://dns.example.com --token eyJ... zones list`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
}

func main() {
	rootCmd.PersistentFlags().String("server", "", "URL geo-dns сервера, напр. http://localhost:8080 (перекрывает config)")
	rootCmd.PersistentFlags().String("token", "", "JWT токен (перекрывает config)")

	viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(zonesCmd)
	rootCmd.AddCommand(whitelistCmd)
	rootCmd.AddCommand(geoCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
