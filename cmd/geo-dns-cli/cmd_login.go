package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate and save JWT token",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")

		if password == "" {
			return fmt.Errorf("--password is required")
		}

		client := NewAPIClient(serverURL(), "")
		token, err := client.Login(username, password)
		if err != nil {
			return err
		}

		if err := saveToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Println("Login successful. Token saved.")
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("username", "u", "admin", "Username")
	loginCmd.Flags().StringP("password", "p", "", "Password")
}
