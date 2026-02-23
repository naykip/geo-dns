package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var zonesCmd = &cobra.Command{
	Use:   "zones",
	Short: "Manage DNS zones",
}

var zonesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all zones",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := NewAPIClient(serverURL(), currentToken())
		zones, err := client.GetZones()
		if err != nil {
			return err
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Origin", "GeoTag", "Records", "NS"})
		table.SetBorder(false)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetAlignment(tablewriter.ALIGN_LEFT)

		for _, zoneList := range zones {
			for _, z := range zoneList {
				ns := ""
				for _, rec := range z.Records {
					if rec.Type == "NS" {
						ns = rec.Value
						break
					}
				}
				table.Append([]string{
					z.Origin,
					z.GeoTag,
					strconv.Itoa(len(z.Records)),
					ns,
				})
			}
		}
		table.Render()
		return nil
	},
}

var zoneAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add or update a zone",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		origin, _ := cmd.Flags().GetString("origin")
		geoTag, _ := cmd.Flags().GetString("geo-tag")

		var z Zone
		if file != "" {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("cannot read file: %w", err)
			}
			if err := json.Unmarshal(data, &z); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
		} else if origin != "" {
			z.Origin = origin
			z.GeoTag = geoTag
			if z.GeoTag == "" {
				z.GeoTag = "default"
			}
		} else {
			return fmt.Errorf("either --file or --origin must be specified")
		}

		client := NewAPIClient(serverURL(), currentToken())
		if err := client.AddZone(z); err != nil {
			return err
		}
		fmt.Printf("Zone %s (%s) added/updated successfully.\n", z.Origin, z.GeoTag)
		return nil
	},
}

var zoneDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a zone (not implemented on server)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("zone deletion is not implemented on the server")
	},
}

func init() {
	zoneAddCmd.Flags().StringP("file", "f", "", "Path to zone JSON file")
	zoneAddCmd.Flags().String("origin", "", "Zone origin (e.g. example.com.)")
	zoneAddCmd.Flags().String("geo-tag", "", "GeoTag (ISO country code or 'default')")

	zonesCmd.AddCommand(zonesListCmd)
	zonesCmd.AddCommand(zoneAddCmd)
	zonesCmd.AddCommand(zoneDeleteCmd)
}
