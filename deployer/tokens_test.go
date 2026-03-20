package deployer

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateServiceTokenUsesExpectedClaims(t *testing.T) {
	token, err := generateServiceToken("supersecret", sfuServiceType, "service-123")
	if err != nil {
		t.Fatalf("generateServiceToken returned error: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts = %d, want 3", len(parts))
	}

	var header jwtHeader
	if err := decodeJWTPart(parts[0], &header); err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if header.Algorithm != "HS256" {
		t.Fatalf("header.Algorithm = %q, want HS256", header.Algorithm)
	}
	if header.Type != "JWT" {
		t.Fatalf("header.Type = %q, want JWT", header.Type)
	}

	var claims serviceTokenClaims
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	if claims.Type != sfuServiceType {
		t.Fatalf("claims.Type = %q, want %q", claims.Type, sfuServiceType)
	}
	if claims.ID != "service-123" {
		t.Fatalf("claims.ID = %q, want service-123", claims.ID)
	}
}

func TestWriteSFUTokenOutputJSONIncludesHeaderWhenRequested(t *testing.T) {
	var output strings.Builder
	if err := writeSFUTokenOutput(&output, "json", "service-123", "token-abc", true); err != nil {
		t.Fatalf("writeSFUTokenOutput returned error: %v", err)
	}

	var payload struct {
		ServiceID string `json:"service_id"`
		Token     string `json:"token"`
		Header    string `json:"header"`
	}
	if err := json.Unmarshal([]byte(output.String()), &payload); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if payload.ServiceID != "service-123" {
		t.Fatalf("payload.ServiceID = %q, want service-123", payload.ServiceID)
	}
	if payload.Token != "token-abc" {
		t.Fatalf("payload.Token = %q, want token-abc", payload.Token)
	}
	if payload.Header != "X-Webhook-Token: token-abc" {
		t.Fatalf("payload.Header = %q, want X-Webhook-Token: token-abc", payload.Header)
	}
}

func decodeJWTPart(part string, dest any) error {
	raw, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}
