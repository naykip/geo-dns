package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type GeoService struct {
	storage *MemoryStorage
	mu      sync.Mutex
}

func NewGeoService(s *MemoryStorage) *GeoService {
	return &GeoService{storage: s}
}

// DownloadAndLoadDB скачивает базу и обновляет её в Storage
func (g *GeoService) DownloadAndLoadDB(url string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	log.Printf("Starting GeoIP update from: %s", url)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	dbPath := "data/geo-db.mmdb"
	_ = os.MkdirAll("data", 0755)

	// Распаковка GZIP на лету
	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip reader error: %w", err)
	}
	defer gzr.Close()

	out, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("file creation error: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, gzr)
	if err != nil {
		return fmt.Errorf("copy error: %w", err)
	}

	// Перезагружаем в Storage
	return g.storage.ReloadGeoDB(dbPath)
}
