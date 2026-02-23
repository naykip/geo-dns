package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var geoCmd = &cobra.Command{
	Use:   "geo",
	Short: "Управление базой данных GeoIP",
	Long: `Команды для обновления базы MaxMind GeoIP2, используемой для geo-тегирования DNS запросов.`,
}

var geoUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Скачать и перезагрузить базу GeoIP",
	Long: `Загружает файл .mmdb.gz по URL, распаковывает и горячо перезагружает
базу GeoIP без перезапуска сервера.

Файл сохраняется как data/geo-db.mmdb на сервере.

Примеры:
  # MaxMind GeoLite2 (требует регистрации на maxmind.com)
  geo-dns-cli geo update --url https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=YOUR_KEY&suffix=tar.gz

  # Собственный хостинг базы
  geo-dns-cli geo update --url https://files.example.com/GeoLite2-Country.mmdb.gz`,
	RunE: func(cmd *cobra.Command, args []string) error {
		geoURL, _ := cmd.Flags().GetString("url")
		if geoURL == "" {
			return fmt.Errorf("--url обязателен: URL до .mmdb.gz файла")
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
	geoUpdateCmd.Flags().String("url", "", "URL до .mmdb.gz файла (MaxMind GeoLite2/GeoIP2)")
	geoCmd.AddCommand(geoUpdateCmd)
}
