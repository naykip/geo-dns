package main

// @title Geo-DNS API
// @version 1.0
// @description API for managing Geo-tagged DNS zones.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Введите 'Bearer <ваш_токен>'

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

type apiHandler struct {
	s       *MemoryStorage
	geo     *GeoService
	metrics *Metrics
}

func NewAPIHandler(s *MemoryStorage, geo *GeoService, m *Metrics) *apiHandler {
	return &apiHandler{s: s, geo: geo, metrics: m}
}

func (h *apiHandler) metricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if h.metrics == nil {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			elapsed := time.Since(start)
			routePattern := chi.RouteContext(r.Context()).RoutePattern()
			if routePattern == "" {
				routePattern = r.URL.Path
			}
			statusCode := strconv.Itoa(ww.Status())
			method := r.Method

			h.metrics.APIRequestsTotal.WithLabelValues(method, routePattern, statusCode).Inc()
			h.metrics.APIRequestDuration.WithLabelValues(method, routePattern).Observe(elapsed.Seconds())
		})
	}
}

func (h *apiHandler) RegisterRoutes(r chi.Router, tokenAuth *jwtauth.JWTAuth, kc *KeycloakValidator) {
	r.Use(h.metricsMiddleware())

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	if h.metrics != nil {
		r.Handle("/metrics", promhttp.HandlerFor(h.metrics.Registry, promhttp.HandlerOpts{}))
	}

	if kc != nil {
		// Режим Keycloak: токены выдаются самим Keycloak
		r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Keycloak auth is enabled. Obtain a token from your Keycloak server.", http.StatusGone)
		})
		r.Group(func(r chi.Router) {
			r.Use(kc.Middleware())
			h.registerProtectedRoutes(r)
		})
	} else {
		// Режим локального JWT: Basic Auth → JWT
		r.Get("/login", h.getLogin)
		r.Group(func(r chi.Router) {
			r.Use(jwtauth.Verifier(tokenAuth))
			r.Use(jwtauth.Authenticator(tokenAuth))
			h.registerProtectedRoutes(r)
		})
	}
}

func (h *apiHandler) registerProtectedRoutes(r chi.Router) {
	r.Get("/zones", h.getZones)
	r.Post("/zone", h.postZone)
	r.Post("/admin/allow", h.postAllowIP)
	r.Post("/geo/update", h.postUpdateGeo)
}

// @Summary Get All Zones
// @Tags zones
// @Security ApiKeyAuth
// @Router /zones [get]
func (h *apiHandler) getZones(w http.ResponseWriter, r *http.Request) {
	h.s.mu.RLock()
	defer h.s.mu.RUnlock()
	json.NewEncoder(w).Encode(h.s.zones)
}

// @Summary Allow CIDR for Recursion
// @Tags admin
// @Param cidr query string true "1.2.3.4/32"
// @Security ApiKeyAuth
// @Router /admin/allow [post]
func (h *apiHandler) postAllowIP(w http.ResponseWriter, r *http.Request) {
	cidr := r.URL.Query().Get("cidr")
	if err := h.s.AddAllowedIP(cidr); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Write([]byte("Added to WhiteList"))
}

// @Summary Login to get JWT
// @Description Basic Auth: username=admin, password=ADMIN_PASSWORD env var
// @Success 200 {string} string "JWT Token"
// @Failure 401 {string} string "Unauthorized"
// @Router /login [get]
func (h *apiHandler) getLogin(w http.ResponseWriter, r *http.Request) {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		http.Error(w, "ADMIN_PASSWORD not set", http.StatusInternalServerError)
		return
	}
	_, pass, ok := r.BasicAuth()
	if !ok || pass != password {
		w.Header().Set("WWW-Authenticate", `Basic realm="geo-dns"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_, tokenString, _ := tokenAuth.Encode(map[string]interface{}{
		"user_id": "admin",
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	w.Write([]byte(tokenString))
}

// @Summary Add or Update Zone
// @Description Creates a new DNS zone or updates an existing one with a geo-tag
// @Tags zones
// @Accept json
// @Produce json
// @Param zone body Zone true "Zone Data"
// @Success 201 {string} string "Zone updated"
// @Security ApiKeyAuth
// @Router /zone [post]
func (h *apiHandler) postZone(w http.ResponseWriter, r *http.Request) {
	var z Zone
	if err := json.NewDecoder(r.Body).Decode(&z); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	h.s.AddZone(z)
	w.WriteHeader(201)
	w.Write([]byte("Zone updated/added successfully"))
}

// @Summary Update GeoIP Database
// @Description Triggers a download and reload of the MaxMind database
// @Tags geo
// @Accept json
// @Produce json
// @Param url query string true "URL to .mmdb.gz file"
// @Success 200 {string} string "GeoDB updated"
// @Security ApiKeyAuth
// @Router /geo/update [post]
func (h *apiHandler) postUpdateGeo(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url query parameter is required", http.StatusBadRequest)
		return
	}

	if err := h.geo.DownloadAndLoadDB(url); err != nil {
		log.Printf("Geo update error: %v", err)
		http.Error(w, "Failed to update GeoIP: "+err.Error(), 500)
		return
	}
	w.Write([]byte("GeoIP database updated and loaded in RAM"))
}
