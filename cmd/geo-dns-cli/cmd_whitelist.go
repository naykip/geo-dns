package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var whitelistCmd = &cobra.Command{
	Use:   "whitelist",
	Short: "Manage DNS recursion whitelist",
}

var whitelistAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a CIDR to the recursion whitelist",
	RunE: func(cmd *cobra.Command, args []string) error {
		cidr, _ := cmd.Flags().GetString("cidr")
		if cidr == "" {
			return fmt.Errorf("--cidr is required")
		}
		client := NewAPIClient(serverURL(), currentToken())
		if err := client.AddWhitelist(cidr); err != nil {
			return err
		}
		fmt.Printf("CIDR %s added to whitelist.\n", cidr)
		return nil
	},
}

func init() {
	whitelistAddCmd.Flags().String("cidr", "", "CIDR to allow (e.g. 1.2.3.4/32)")
	whitelistCmd.AddCommand(whitelistAddCmd)
}
