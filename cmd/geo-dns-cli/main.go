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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
}

func main() {
	rootCmd.PersistentFlags().String("server", "", "geo-dns server URL (overrides config)")
	rootCmd.PersistentFlags().String("token", "", "JWT token (overrides config)")

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
