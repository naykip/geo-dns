package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SOAData struct {
	Ns      string `json:"ns"`
	Mbox    string `json:"mbox"`
	Serial  uint32 `json:"serial"`
	Refresh uint32 `json:"refresh"`
	Retry   uint32 `json:"retry"`
	Expire  uint32 `json:"expire"`
	MinTTL  uint32 `json:"min_ttl"`
}

type ResourceRecord struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   uint32 `json:"ttl"`
}

type Zone struct {
	Origin  string           `json:"origin"`
	GeoTag  string           `json:"geo_tag"`
	SOA     SOAData          `json:"soa"`
	Records []ResourceRecord `json:"records"`
}

type APIClient struct {
	base   string
	token  string
	client *http.Client
}

func NewAPIClient(base, token string) *APIClient {
	return &APIClient{
		base:  strings.TrimRight(base, "/"),
		token: token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *APIClient) do(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.client.Do(req)
}

func (c *APIClient) Login(username, password string) (string, error) {
	req, err := http.NewRequest("GET", c.base+"/login", nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(username, password)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return strings.TrimSpace(string(data)), nil
}

func (c *APIClient) GetZones() (map[string][]Zone, error) {
	resp, err := c.do("GET", "/zones", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get zones failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var result map[string][]Zone
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *APIClient) AddZone(z Zone) error {
	data, err := json.Marshal(z)
	if err != nil {
		return err
	}
	resp, err := c.do("POST", "/zone", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add zone failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *APIClient) AddWhitelist(cidr string) error {
	resp, err := c.do("POST", "/admin/allow?cidr="+url.QueryEscape(cidr), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("whitelist failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *APIClient) UpdateGeo(geoURL string) error {
	resp, err := c.do("POST", "/geo/update?url="+url.QueryEscape(geoURL), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("geo update failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
