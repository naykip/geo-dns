package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// KeycloakValidator проверяет JWT-токены Keycloak через JWKS endpoint.
// Ключи обновляются автоматически каждые 5 минут.
type KeycloakValidator struct {
	jwksURL  string
	issuer   string
	audience string // опционально: KEYCLOAK_AUDIENCE
	keySet   jwk.Set
	mu       sync.RWMutex
}

// NewKeycloakValidator инициализирует валидатор и сразу загружает JWKS.
// audience — необязательный параметр; если пустой, проверка aud пропускается.
func NewKeycloakValidator(baseURL, realm, audience string) (*KeycloakValidator, error) {
	if baseURL == "" || realm == "" {
		return nil, fmt.Errorf("KEYCLOAK_URL and KEYCLOAK_REALM must be set")
	}
	kv := &KeycloakValidator{
		jwksURL:  fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", baseURL, realm),
		issuer:   fmt.Sprintf("%s/realms/%s", baseURL, realm),
		audience: audience,
	}
	if err := kv.fetchKeys(); err != nil {
		return nil, fmt.Errorf("keycloak JWKS fetch: %w", err)
	}
	go kv.autoRefresh()
	log.Printf("Keycloak auth enabled: issuer=%s", kv.issuer)
	return kv, nil
}

func (kv *KeycloakValidator) fetchKeys() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	set, err := jwk.Fetch(ctx, kv.jwksURL)
	if err != nil {
		return err
	}
	kv.mu.Lock()
	kv.keySet = set
	kv.mu.Unlock()
	return nil
}

func (kv *KeycloakValidator) autoRefresh() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if err := kv.fetchKeys(); err != nil {
			log.Printf("keycloak JWKS refresh error: %v", err)
		}
	}
}

// Middleware возвращает chi-совместимый middleware для проверки Keycloak Bearer токенов.
func (kv *KeycloakValidator) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")

			kv.mu.RLock()
			keySet := kv.keySet
			kv.mu.RUnlock()

			opts := []jwt.ParseOption{
				jwt.WithKeySet(keySet),
				jwt.WithValidate(true),
				jwt.WithIssuer(kv.issuer),
			}
			if kv.audience != "" {
				opts = append(opts, jwt.WithAudience(kv.audience))
			}

			if _, err := jwt.Parse([]byte(tokenStr), opts...); err != nil {
				http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
