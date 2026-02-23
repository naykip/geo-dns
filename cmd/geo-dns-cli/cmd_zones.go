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
	Short: "Управление DNS зонами",
	Long: `Команды для просмотра и управления DNS зонами на сервере.

Подкоманды:
  list    — показать все зоны в виде таблицы
  add     — добавить или обновить зону из JSON файла или флагов
  delete  — заглушка (удаление не реализовано на сервере)`,
}

var zonesListCmd = &cobra.Command{
	Use:   "list",
	Short: "Показать все зоны",
	Long: `Выводит все DNS зоны сервера в виде таблицы.

Колонки: Origin | GeoTag | Records (кол-во) | NS

Примеры:
  geo-dns-cli zones list
  geo-dns-cli --server http://10.0.0.1:8080 zones list`,
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
	Short: "Добавить или обновить зону",
	Long: `Добавляет новую зону или обновляет существующую (по origin + geo_tag).

Приоритет: --file > --origin/--geo-tag

Структура JSON файла:
  {
    "origin":  "example.com.",      // FQDN с точкой на конце
    "geo_tag": "RU",                // ISO код страны или "default"
    "soa": { "ns": "ns1.example.com.", "mbox": "admin.example.com.",
             "serial": 2024010101, "refresh": 86400, "retry": 7200,
             "expire": 3600000, "min_ttl": 300 },
    "records": [
      { "name": "example.com.", "type": "A", "value": "1.2.3.4", "ttl": 300 }
    ]
  }

Типы записей: A, AAAA, CNAME, NS, MX, TXT
Формат MX value: "10 mail.example.com."

Примеры:
  # Из файла
  geo-dns-cli zones add --file zone-ru.json
  geo-dns-cli zones add -f examples/zone-default.json

  # Только origin (пустая зона, записи добавить позже)
  geo-dns-cli zones add --origin example.com. --geo-tag RU`,
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
	zoneAddCmd.Flags().StringP("file", "f", "", "Путь к JSON файлу зоны")
	zoneAddCmd.Flags().String("origin", "", "Origin зоны, напр. example.com. (с точкой)")
	zoneAddCmd.Flags().String("geo-tag", "", "GeoTag: ISO код страны (RU, DE, US...) или 'default'")

	zonesCmd.AddCommand(zonesListCmd)
	zonesCmd.AddCommand(zoneAddCmd)
	zonesCmd.AddCommand(zoneDeleteCmd)
}
