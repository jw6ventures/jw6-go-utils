package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	jwt "github.com/go-jose/go-jose/v4/jwt"
)

func testKey(t *testing.T, size int) []byte {
	t.Helper()
	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return key
}

func TestCookieCodecRoundTrip(t *testing.T) {
	codec, err := NewCookieCodec(testKey(t, 32))
	if err != nil {
		t.Fatalf("NewCookieCodec: %v", err)
	}

	type payload struct {
		State string `json:"state"`
		Nonce string `json:"nonce"`
	}
	in := payload{State: "abc", Nonce: "xyz"}

	sealed, err := codec.Seal(in, time.Hour)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	var out payload
	if err := codec.Open(sealed, &out); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if out != in {
		t.Fatalf("round trip mismatch: got %+v want %+v", out, in)
	}
}

func TestCookieCodecKeySizes(t *testing.T) {
	for _, size := range []int{16, 24, 32} {
		if _, err := NewCookieCodec(testKey(t, size)); err != nil {
			t.Errorf("size %d: unexpected error %v", size, err)
		}
	}
	if _, err := NewCookieCodec(testKey(t, 10)); err == nil {
		t.Errorf("expected error for invalid key size")
	}
}

func TestCookieCodecTamperDetected(t *testing.T) {
	codec, err := NewCookieCodec(testKey(t, 32))
	if err != nil {
		t.Fatalf("NewCookieCodec: %v", err)
	}

	sealed, err := codec.Seal(map[string]string{"k": "v"}, time.Hour)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	// Flip the last character to corrupt the ciphertext/tag.
	tampered := sealed[:len(sealed)-1]
	if sealed[len(sealed)-1] == 'A' {
		tampered += "B"
	} else {
		tampered += "A"
	}

	var out map[string]string
	if err := codec.Open(tampered, &out); err == nil {
		t.Fatalf("expected authentication failure for tampered value")
	}
}

func TestCookieCodecWrongKey(t *testing.T) {
	a, _ := NewCookieCodec(testKey(t, 32))
	b, _ := NewCookieCodec(testKey(t, 32))

	sealed, err := a.Seal(map[string]string{"k": "v"}, time.Hour)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	var out map[string]string
	if err := b.Open(sealed, &out); err == nil {
		t.Fatalf("expected failure opening with a different key")
	}
}

func TestCookieCodecExpiry(t *testing.T) {
	codec, _ := NewCookieCodec(testKey(t, 32))
	sealed, err := codec.Seal(map[string]string{"k": "v"}, -time.Second)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	// ttl <= 0 means "no embedded expiry", so this must still open.
	var out map[string]string
	if err := codec.Open(sealed, &out); err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Force an expiry in the past via the envelope directly.
	expired, err := codec.Seal(map[string]string{"k": "v"}, time.Millisecond)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := codec.Open(expired, &out); err == nil {
		t.Fatalf("expected expiry error")
	}
}

func TestSetAndReadCookie(t *testing.T) {
	codec, _ := NewCookieCodec(testKey(t, 32))

	rec := httptest.NewRecorder()
	type sess struct {
		Sub string `json:"sub"`
	}
	if err := codec.SetCookie(rec, "session", sess{Sub: "user-1"}, time.Hour, DefaultCookieOptions()); err != nil {
		t.Fatalf("SetCookie: %v", err)
	}

	setCookie := rec.Result().Cookies()
	if len(setCookie) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(setCookie))
	}
	if !setCookie[0].HttpOnly || !setCookie[0].Secure {
		t.Errorf("expected hardened cookie attributes")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(setCookie[0])

	var out sess
	if err := codec.ReadCookie(req, "session", &out); err != nil {
		t.Fatalf("ReadCookie: %v", err)
	}
	if out.Sub != "user-1" {
		t.Fatalf("got %q want user-1", out.Sub)
	}
}

func TestCookieNameBoundToSealedValue(t *testing.T) {
	codec, _ := NewCookieCodec(testKey(t, 32))

	rec := httptest.NewRecorder()
	if err := codec.SetCookie(rec, "login", map[string]string{"state": "abc"}, time.Hour, DefaultCookieOptions()); err != nil {
		t.Fatalf("SetCookie: %v", err)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	replayed := *cookies[0]
	replayed.Name = "session"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&replayed)

	var out map[string]string
	if err := codec.ReadCookie(req, "session", &out); err == nil {
		t.Fatalf("expected cookie authentication to fail under a different name")
	}
}

func TestResolveScopes(t *testing.T) {
	if got := resolveScopes(nil); len(got) == 0 || got[0] != "openid" {
		t.Fatalf("default scopes missing openid: %v", got)
	}
	got := resolveScopes([]string{"email"})
	if got[0] != "openid" {
		t.Fatalf("expected openid to be prepended, got %v", got)
	}
	got = resolveScopes([]string{"openid", "groups"})
	if len(got) != 2 {
		t.Fatalf("expected openid not duplicated, got %v", got)
	}
}

// mockProvider is a minimal OIDC issuer for end-to-end testing of the
// authorization-code flow: discovery, token endpoint and JWKS.
type mockProvider struct {
	server   *httptest.Server
	signer   jose.Signer
	clientID string
	nonce    string
	subject  string
}

func newMockProvider(t *testing.T, clientID string) *mockProvider {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	const kid = "test-key"
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: priv},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", kid),
	)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	mp := &mockProvider{signer: signer, clientID: clientID, subject: "subject-123"}

	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key:       priv.Public(),
		KeyID:     kid,
		Algorithm: "RS256",
		Use:       "sig",
	}}}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		base := mp.server.URL
		writeJSON(w, map[string]any{
			"issuer":                 base,
			"authorization_endpoint": base + "/authorize",
			"token_endpoint":         base + "/token",
			"jwks_uri":               base + "/jwks",
			"userinfo_endpoint":      base + "/userinfo",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, jwks)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		idToken := mp.signIDToken(t)
		writeJSON(w, map[string]any{
			"access_token": "access-token-value",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     idToken,
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"sub":   mp.subject,
			"email": "user@example.com",
		})
	})

	mp.server = httptest.NewServer(mux)
	t.Cleanup(mp.server.Close)
	return mp
}

func (mp *mockProvider) signIDToken(t *testing.T) string {
	t.Helper()
	now := time.Now()
	claims := map[string]any{
		"iss":            mp.server.URL,
		"sub":            mp.subject,
		"aud":            mp.clientID,
		"exp":            now.Add(time.Hour).Unix(),
		"iat":            now.Unix(),
		"nonce":          mp.nonce,
		"email":          "user@example.com",
		"email_verified": true,
		"name":           "Test User",
	}
	raw, err := jwt.Signed(mp.signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatalf("sign id token: %v", err)
	}
	return raw
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func TestAuthRequest(t *testing.T) {
	mp := newMockProvider(t, "client-abc")
	auth, err := NewAuthenticator(context.Background(), Config{
		Issuer:      mp.server.URL,
		ClientID:    "client-abc",
		RedirectURL: "https://app.example.com/callback",
	})
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}

	req, err := auth.AuthRequest()
	if err != nil {
		t.Fatalf("AuthRequest: %v", err)
	}
	if req.State == "" || req.Nonce == "" || req.PKCEVerifier == "" {
		t.Fatalf("expected state/nonce/verifier to be populated: %+v", req)
	}

	u, err := url.Parse(req.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	q := u.Query()
	if q.Get("state") != req.State {
		t.Errorf("state not in url")
	}
	if q.Get("nonce") != req.Nonce {
		t.Errorf("nonce not in url")
	}
	if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		t.Errorf("expected PKCE S256 challenge in url, got %v", q)
	}
	if !strings.Contains(q.Get("scope"), "openid") {
		t.Errorf("expected openid scope, got %q", q.Get("scope"))
	}
}

func TestExchange(t *testing.T) {
	mp := newMockProvider(t, "client-abc")
	auth, err := NewAuthenticator(context.Background(), Config{
		Issuer:      mp.server.URL,
		ClientID:    "client-abc",
		RedirectURL: "https://app.example.com/callback",
	})
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}

	req, err := auth.AuthRequest()
	if err != nil {
		t.Fatalf("AuthRequest: %v", err)
	}
	mp.nonce = req.Nonce

	session, err := auth.Exchange(context.Background(), "auth-code", req.Nonce, req.PKCEVerifier)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if session.Subject != "subject-123" {
		t.Errorf("subject: got %q", session.Subject)
	}
	if session.Email != "user@example.com" || !session.EmailVerified {
		t.Errorf("email claims not parsed: %+v", session)
	}
	if session.Name != "Test User" {
		t.Errorf("name: got %q", session.Name)
	}
	if session.AccessToken != "access-token-value" {
		t.Errorf("access token: got %q", session.AccessToken)
	}
}

func TestExchangeNonceMismatch(t *testing.T) {
	mp := newMockProvider(t, "client-abc")
	auth, err := NewAuthenticator(context.Background(), Config{
		Issuer:      mp.server.URL,
		ClientID:    "client-abc",
		RedirectURL: "https://app.example.com/callback",
	})
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}

	req, _ := auth.AuthRequest()
	mp.nonce = "the-real-nonce"

	if _, err := auth.Exchange(context.Background(), "auth-code", "a-different-nonce", req.PKCEVerifier); err == nil {
		t.Fatalf("expected nonce mismatch error")
	}
}

func TestExchangeRequiresNonceAndPKCEVerifier(t *testing.T) {
	mp := newMockProvider(t, "client-abc")
	auth, err := NewAuthenticator(context.Background(), Config{
		Issuer:      mp.server.URL,
		ClientID:    "client-abc",
		RedirectURL: "https://app.example.com/callback",
	})
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}

	req, err := auth.AuthRequest()
	if err != nil {
		t.Fatalf("AuthRequest: %v", err)
	}

	if _, err := auth.Exchange(context.Background(), "auth-code", "", req.PKCEVerifier); err == nil {
		t.Fatalf("expected missing nonce error")
	}
	if _, err := auth.Exchange(context.Background(), "auth-code", req.Nonce, ""); err == nil {
		t.Fatalf("expected missing PKCE verifier error")
	}
}

func TestNewAuthenticatorValidation(t *testing.T) {
	cases := []Config{
		{ClientID: "c", RedirectURL: "r"},       // missing issuer
		{Issuer: "https://x", RedirectURL: "r"}, // missing client id
		{Issuer: "https://x", ClientID: "c"},    // missing redirect
	}
	for i, c := range cases {
		if _, err := NewAuthenticator(context.Background(), c); err == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
}
