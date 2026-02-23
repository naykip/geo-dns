package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var whitelistCmd = &cobra.Command{
	Use:   "whitelist",
	Short: "Управление whitelist рекурсивных DNS запросов",
	Long: `Управляет списком IP-адресов (CIDR), которым разрешена рекурсия через 8.8.8.8.

По умолчанию рекурсия разрешена только с 127.0.0.1/32.
Остальным клиентам возвращается REFUSED для неизвестных имён.`,
}

var whitelistAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Добавить CIDR в whitelist рекурсии",
	Long: `Разрешает рекурсивные DNS запросы для указанного CIDR диапазона.

Примеры:
  # Разрешить один IP
  geo-dns-cli whitelist add --cidr 203.0.113.5/32

  # Разрешить подсеть (офис, VPN)
  geo-dns-cli whitelist add --cidr 10.0.0.0/8
  geo-dns-cli whitelist add --cidr 192.168.1.0/24`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cidr, _ := cmd.Flags().GetString("cidr")
		if cidr == "" {
			return fmt.Errorf("--cidr обязателен, напр. --cidr 1.2.3.4/32")
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
	whitelistAddCmd.Flags().String("cidr", "", "CIDR для разрешения рекурсии, напр. 1.2.3.4/32")
	whitelistCmd.AddCommand(whitelistAddCmd)
}
