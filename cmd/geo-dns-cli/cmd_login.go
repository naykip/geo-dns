package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Аутентификация и сохранение JWT токена",
	Long: `Получает JWT токен через Basic Auth (GET /login) и сохраняет в конфиг.

Токен действует 24 часа. После истечения повторите login.
Конфиг: ~/.config/geo-dns-cli/config.yaml (права 0600)

Примеры:
  # Сервер уже в конфиге — просто логин
  geo-dns-cli login -p mypassword

  # Первый раз — указать сервер явно
  geo-dns-cli --server http://localhost:8080 login -p mypassword

  # Удалённый сервер по HTTPS
  geo-dns-cli --server https://dns.example.com login -p mypassword

  # Другой username
  geo-dns-cli login -u operator -p mypassword`,
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")

		if password == "" {
			return fmt.Errorf("--password / -p обязателен")
		}

		client := NewAPIClient(serverURL(), "")
		token, err := client.Login(username, password)
		if err != nil {
			return err
		}

		if err := saveToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Printf("Успешный вход. Сервер: %s\nТокен сохранён в ~/.config/geo-dns-cli/config.yaml\n", serverURL())
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("username", "u", "admin", "Имя пользователя (default: admin)")
	loginCmd.Flags().StringP("password", "p", "", "Пароль (переменная ADMIN_PASSWORD на сервере)")
}
