// ACC-E03 T5b — test representativo de la aceptación dual en la flota, con la config REAL del
// servicio. Lleva el tag `fleetdual` porque necesita la go-shared con el keyring (local, sin
// release todavía): con GOWORK=off el go.mod fija la versión vieja, HMAC-only. Correr en modo
// workspace: `go test -tags fleetdual -run FleetDual .`. Al bumpear go-shared, quitar el tag.

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	sharedmw "github.com/hornosg/go-shared/infrastructure/middleware"
)

const fleetDualSecret = "fleet-dual-test-secret-min-32-chars-xxxxxx"

func fleetDualKeyring(t *testing.T) (*rsa.PrivateKey, *rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if err := os.WriteFile(filepath.Join(dir, sharedmw.KeyringKID(&key.PublicKey)+".pem"), pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	// El wiring real no pasa JWTPublicKeyDir: go-shared lo toma de JWT_PUBLIC_KEY_DIR.
	t.Setenv("JWT_SECRET", fleetDualSecret)
	t.Setenv("JWT_PUBLIC_KEY_DIR", dir)
	return key, other, string(pemBytes)
}

func fleetDualClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"tenant_id": "tenant-a",
		"user_id":   "user-123",
		"namespace": "mc",
		"roles":     []string{"system_admin"},
		"iss":       "iam-service",
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
	}
}

func fleetDualSign(t *testing.T, m jwt.SigningMethod, key interface{}, kid string) string {
	t.Helper()
	tok := jwt.NewWithClaims(m, fleetDualClaims())
	if kid != "" {
		tok.Header["kid"] = kid
	}
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func fleetDualDo(r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", "tenant-a")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// fleetDualAssert: RS256 válido (kid del keyring) → 200 con claims poblados; otra clave, alg:none
// y HS256 con la pública como secreto → 401.
func fleetDualAssert(t *testing.T, r *gin.Engine, path string, key, other *rsa.PrivateKey, pubPEM string) {
	t.Helper()
	kid := sharedmw.KeyringKID(&key.PublicKey)
	if w := fleetDualDo(r, path, fleetDualSign(t, jwt.SigningMethodRS256, key, kid)); w.Code != http.StatusOK || w.Body.String() != "user-123" {
		t.Fatalf("RS256 válido: code=%d body=%q", w.Code, w.Body.String())
	}
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, fleetDualClaims()).SignedString(jwt.UnsafeAllowNoneSignatureType)
	for name, tok := range map[string]string{
		"RS256 firmado con OTRA clave": fleetDualSign(t, jwt.SigningMethodRS256, other, kid),
		"alg:none":                     none,
		"HS256 con la pública como secreto (RS/HS)": fleetDualSign(t, jwt.SigningMethodHS256, []byte(pubPEM), kid),
	} {
		if w := fleetDualDo(r, path, tok); w.Code != http.StatusUnauthorized {
			t.Errorf("%s: code=%d, se espera 401", name, w.Code)
		}
	}
}

func claimsUserID(c *gin.Context) {
	v, _ := c.Get("jwt_claims")
	mc, _ := v.(jwt.MapClaims)
	uid, _ := mc["user_id"].(string)
	c.String(http.StatusOK, uid)
}

func TestFleetDual_Sales_TenantValidation(t *testing.T) {
	key, other, pubPEM := fleetDualKeyring(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(sharedmw.TenantValidation(tenantValidationConfig()))
	r.GET("/api/v1/pos/probe", claimsUserID)
	fleetDualAssert(t, r, "/api/v1/pos/probe", key, other, pubPEM)
}
