package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	_ "geo-dns/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/miekg/dns"
)

var tokenAuth *jwtauth.JWTAuth

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "super-secret-key"
	}
	tokenAuth = jwtauth.New("HS256", []byte(secret), nil)
}

func main() {
	storage := NewStorage()
	geoService := NewGeoService(storage)
	api := NewAPIHandler(storage, geoService)

	apiPort := strings.TrimPrefix(os.Getenv("API_PORT"), ":")
	if apiPort == "" {
		apiPort = "8080"
	}
	dnsPort := strings.TrimPrefix(os.Getenv("DNS_PORT"), ":")
	if dnsPort == "" {
		dnsPort = "53"
	}

	// DNS Server (UDP)
	go func() {
		dnsHandler := &DNSHandler{Storage: storage}
		server := &dns.Server{Addr: ":" + dnsPort, Net: "udp", Handler: dnsHandler}
		log.Printf("Starting DNS UDP on :%s", dnsPort)
		server.ListenAndServe()
	}()

	// DNS Server (TCP)
	go func() {
		dnsHandler := &DNSHandler{Storage: storage}
		server := &dns.Server{Addr: ":" + dnsPort, Net: "tcp", Handler: dnsHandler}
		log.Printf("Starting DNS TCP on :%s", dnsPort)
		server.ListenAndServe()
	}()

	// Keycloak (опционально)
	var kc *KeycloakValidator
	if os.Getenv("KEYCLOAK_ENABLED") == "true" {
		var err error
		kc, err = NewKeycloakValidator(
			os.Getenv("KEYCLOAK_URL"),
			os.Getenv("KEYCLOAK_REALM"),
			os.Getenv("KEYCLOAK_AUDIENCE"),
		)
		if err != nil {
			log.Fatalf("Keycloak init failed: %v", err)
		}
	}

	r := chi.NewRouter()
	api.RegisterRoutes(r, tokenAuth, kc)

	useSSL := os.Getenv("API_SSL") == "true"
	if useSSL {
		cert, key := os.Getenv("SSL_CERT_PATH"), os.Getenv("SSL_KEY_PATH")
		if _, err := os.Stat(cert); os.IsNotExist(err) {
			log.Printf("SSL cert not found, using HTTP on :%s", apiPort)
			log.Fatal(http.ListenAndServe(":"+apiPort, r))
		} else {
			log.Printf("Starting HTTPS on :%s", apiPort)
			log.Fatal(http.ListenAndServeTLS(":"+apiPort, cert, key, r))
		}
	} else {
		log.Printf("Starting HTTP on :%s", apiPort)
		log.Fatal(http.ListenAndServe(":"+apiPort, r))
	}
}
