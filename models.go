package main

// SOAData представляет запись Start of Authority (паспорт зоны)
type SOAData struct {
	Ns      string `json:"ns" example:"ns1.example.com."`
	Mbox    string `json:"mbox" example:"admin.example.com."`
	Serial  uint32 `json:"serial" example:"2024052501"`
	Refresh uint32 `json:"refresh" example:"86400"`
	Retry   uint32 `json:"retry" example:"7200"`
	Expire  uint32 `json:"expire" example:"3600000"`
	MinTTL  uint32 `json:"minttl" example:"172800"`
}

// ResourceRecord представляет одну DNS-запись (A, MX, TXT и т.д.)
type ResourceRecord struct {
	Name  string `json:"name" example:"example.com."`
	Type  string `json:"type" example:"A"`
	Value string `json:"value" example:"1.2.3.4"`
	TTL   uint32 `json:"ttl" example:"3600"`
}

// Zone объединяет настройки зоны для конкретного Geo-тега
type Zone struct {
	Origin  string           `json:"origin" example:"example.com."`
	GeoTag  string           `json:"geo_tag" example:"RU"`
	SOA     SOAData          `json:"soa"`
	Records []ResourceRecord `json:"records"`
}
