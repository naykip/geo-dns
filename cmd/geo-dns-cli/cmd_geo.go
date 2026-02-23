package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var geoCmd = &cobra.Command{
	Use:   "geo",
	Short: "Manage GeoIP database",
}

var geoUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download and reload GeoIP database",
	RunE: func(cmd *cobra.Command, args []string) error {
		geoURL, _ := cmd.Flags().GetString("url")
		if geoURL == "" {
			return fmt.Errorf("--url is required")
		}
		client := NewAPIClient(serverURL(), currentToken())
		if err := client.UpdateGeo(geoURL); err != nil {
			return err
		}
		fmt.Println("GeoIP database updated successfully.")
		return nil
	},
}

func init() {
	geoUpdateCmd.Flags().String("url", "", "URL to .mmdb.gz file")
	geoCmd.AddCommand(geoUpdateCmd)
}
