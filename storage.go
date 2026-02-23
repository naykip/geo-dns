package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

const (
	dbPath        = "data/zones.json"
	whitelistPath = "data/whitelist.json"
)

type MemoryStorage struct {
	mu         sync.RWMutex
	zones      map[string][]Zone
	allowedNet []string
	geoReader  *geoip2.Reader
}

func NewStorage() *MemoryStorage {
	s := &MemoryStorage{
		zones:      make(map[string][]Zone),
		allowedNet: []string{"127.0.0.1/32"},
	}
	s.loadFromDisk()
	s.loadWhitelist()
	return s
}

func (s *MemoryStorage) IsAllowed(ip net.IP) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, cidr := range s.allowedNet {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *MemoryStorage) AddAllowedIP(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.allowedNet = append(s.allowedNet, cidr)
	snapshot := make([]string, len(s.allowedNet))
	copy(snapshot, s.allowedNet)
	s.mu.Unlock()

	data, _ := json.Marshal(snapshot)
	if err := os.WriteFile(whitelistPath, data, 0644); err != nil {
		log.Printf("saveWhitelist error: %v", err)
	}
	return nil
}

func (s *MemoryStorage) AddZone(z Zone) {
	s.mu.Lock()
	origin := strings.ToLower(z.Origin)
	if !strings.HasSuffix(origin, ".") {
		origin += "."
	}
	z.Origin = origin

	zones := s.zones[origin]
	updated := false
	for i, ez := range zones {
		if strings.EqualFold(ez.GeoTag, z.GeoTag) {
			s.zones[origin][i] = z
			updated = true
			break
		}
	}
	if !updated {
		s.zones[origin] = append(s.zones[origin], z)
	}
	// Снимаем snapshot под write-lock, чтобы не было гонки
	data, _ := json.MarshalIndent(s.zones, "", "  ")
	s.mu.Unlock()

	_ = os.MkdirAll("data", 0755)
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		log.Printf("saveToDisk error: %v", err)
	}
}

func (s *MemoryStorage) loadFromDisk() {
	file, err := os.ReadFile(dbPath)
	if err == nil {
		s.mu.Lock()
		_ = json.Unmarshal(file, &s.zones)
		s.mu.Unlock()
	}
}

func (s *MemoryStorage) loadWhitelist() {
	file, err := os.ReadFile(whitelistPath)
	if err == nil {
		_ = json.Unmarshal(file, &s.allowedNet)
	}
}

func (s *MemoryStorage) GetGeoTag(ipStr string) string {
	s.mu.RLock()
	reader := s.geoReader
	s.mu.RUnlock()
	if reader == nil {
		return "default"
	}
	ip := net.ParseIP(ipStr)
	record, err := reader.Country(ip)
	if err != nil || record.Country.IsoCode == "" {
		return "default"
	}
	return record.Country.IsoCode
}

// GetRecordsForQuery ищет записи с учётом Geo-приоритета.
// Сначала по конкретному тегу, если нет — берёт "default".
func (s *MemoryStorage) GetRecordsForQuery(name string, geoTag string) []ResourceRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	name = strings.ToLower(name)

	for _, tag := range []string{geoTag, "default"} {
		// Прямой поиск по ключу map вместо перебора всех зон
		for origin, zones := range s.zones {
			if !strings.HasSuffix(name, origin) && name != origin {
				continue
			}
			for _, zone := range zones {
				if !strings.EqualFold(zone.GeoTag, tag) {
					continue
				}
				var result []ResourceRecord
				for _, rec := range zone.Records {
					if strings.ToLower(rec.Name) == name {
						result = append(result, rec)
					}
				}
				if len(result) > 0 {
					return result
				}
			}
		}
	}
	return nil
}

func (s *MemoryStorage) GetSOA(name string, geoTag string) *SOAData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	name = strings.ToLower(name)
	for _, zones := range s.zones {
		for _, zone := range zones {
			if strings.HasSuffix(name, strings.ToLower(zone.Origin)) {
				if strings.EqualFold(zone.GeoTag, geoTag) || zone.GeoTag == "default" {
					return &zone.SOA
				}
			}
		}
	}
	return nil
}

// GetZone возвращает зону по origin с учётом geo-приоритета (тег → "default").
func (s *MemoryStorage) GetZone(origin, geoTag string) *Zone {
	s.mu.RLock()
	defer s.mu.RUnlock()
	origin = strings.ToLower(origin)
	for _, tag := range []string{geoTag, "default"} {
		for _, zone := range s.zones[origin] {
			if strings.EqualFold(zone.GeoTag, tag) {
				z := zone
				return &z
			}
		}
	}
	return nil
}

func (s *MemoryStorage) ReloadGeoDB(path string) error {
	newReader, err := geoip2.Open(path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.geoReader != nil {
		s.geoReader.Close()
	}
	s.geoReader = newReader
	s.mu.Unlock()
	return nil
}
