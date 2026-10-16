// ACC-E03 T5b — test de CONTRATO (fase TEST, @dev-qa): criterio (b) pide el token RS256 con el
// **kid de T2**, no con una clave generada. El keyring se arma con la pública REAL de T2 (material
// público, testdata de go-shared). El caso positivo firma con la privada real de T2 (no versionada):
// ACC_E03_T2_PRIVATE_KEY_FILE=$DEVY_PATH/platform/iam-service/keys/jwt_signing_private.pem; sin la
// env, el positivo hace skip y los negativos corren igual.
// Correr: go test -tags fleetdual -run FleetDualT2 <pkg>  (quitar el tag al bumpear go-shared).

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	sharedmw "github.com/hornosg/go-shared/infrastructure/middleware"
)

const fleetDualT2KID = "acc-69a4a91f940a"

func fleetDualT2Router(t *testing.T) (*gin.Engine, []byte) {
	t.Helper()
	pubPEM, err := os.ReadFile(filepath.Join("../..", "libs/go-shared/infrastructure/middleware/testdata", fleetDualT2KID+".pem"))
	if err != nil {
		t.Fatalf("pública de T2: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "t2.pem"), pubPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JWT_SECRET", fleetDualSecret)
	t.Setenv("JWT_PUBLIC_KEY_DIR", dir)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(sharedmw.TenantValidation(tenantValidationConfig()))
	r.GET("/api/v1/probe", claimsUserID)
	return r, pubPEM
}

func TestFleetDualT2_Sales_KidDeT2(t *testing.T) {
	r, pubPEM := fleetDualT2Router(t)

	// Negativos (siempre): kid de T2 firmado con otra clave; confusión RS/HS con la pública de T2.
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for name, tok := range map[string]string{
		"kid de T2 firmado con OTRA clave":        fleetDualSign(t, jwt.SigningMethodRS256, other, fleetDualT2KID),
		"HS256 con la pública de T2 como secreto": fleetDualSign(t, jwt.SigningMethodHS256, pubPEM, fleetDualT2KID),
	} {
		if w := fleetDualDo(r, "/api/v1/probe", tok); w.Code != http.StatusUnauthorized {
			t.Errorf("%s: code=%d, se espera 401", name, w.Code)
		}
	}

	// Positivo con la privada real de T2.
	path := os.Getenv("ACC_E03_T2_PRIVATE_KEY_FILE")
	if path == "" {
		t.Skip("ACC_E03_T2_PRIVATE_KEY_FILE no seteada — positivo con la clave real de T2 no corrido")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("privada de T2: %v", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		t.Fatal("privada de T2: no es PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("privada de T2: %v", err)
	}
	key := parsed.(*rsa.PrivateKey)
	if got := sharedmw.KeyringKID(&key.PublicKey); got != fleetDualT2KID {
		t.Fatalf("kid derivado de la privada = %s, se espera %s", got, fleetDualT2KID)
	}
	if w := fleetDualDo(r, "/api/v1/probe", fleetDualSign(t, jwt.SigningMethodRS256, key, fleetDualT2KID)); w.Code != http.StatusOK || w.Body.String() != "user-123" {
		t.Fatalf("RS256 con la clave real de T2: code=%d body=%q", w.Code, w.Body.String())
	}
}
