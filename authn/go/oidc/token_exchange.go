package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type TokenExchanger struct {
	discovery    *DiscoveryClient
	clientID     string
	clientSecret string
	redirectURL  string
	httpClient   *http.Client
}

type PKCEParams struct {
	CodeVerifier  string
	CodeChallenge string
}

func NewTokenExchanger(discovery *DiscoveryClient, clientID, clientSecret, redirectURL string) *TokenExchanger {
	return &TokenExchanger{
		discovery:    discovery,
		clientID:     strings.TrimSpace(clientID),
		clientSecret: strings.TrimSpace(clientSecret),
		redirectURL:  strings.TrimSpace(redirectURL),
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func GeneratePKCE() (PKCEParams, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return PKCEParams{}, fmt.Errorf("pkce random: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(buf)
	hash := sha256.Sum256([]byte(verifier))
	return PKCEParams{
		CodeVerifier:  verifier,
		CodeChallenge: base64.RawURLEncoding.EncodeToString(hash[:]),
	}, nil
}

func GenerateState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("state random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func GenerateNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("nonce random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (t *TokenExchanger) AuthorizationURL(ctx context.Context, state, nonce string, pkce PKCEParams, scopes []string) (string, error) {
	doc, err := t.discovery.Discover(ctx)
	if err != nil {
		return "", fmt.Errorf("oidc auth url: %w", err)
	}

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", t.clientID)
	params.Set("redirect_uri", t.redirectURL)
	params.Set("scope", strings.Join(scopes, " "))
	params.Set("state", state)
	params.Set("nonce", nonce)
	params.Set("code_challenge", pkce.CodeChallenge)
	params.Set("code_challenge_method", "S256")
	return doc.AuthorizationEndpoint + "?" + params.Encode(), nil
}

func (t *TokenExchanger) ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenResponse, error) {
	doc, err := t.discovery.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("oidc token exchange: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", t.redirectURL)
	form.Set("client_id", t.clientID)
	form.Set("code_verifier", codeVerifier)
	if t.clientSecret != "" {
		form.Set("client_secret", t.clientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, doc.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oidc token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oidc token fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("oidc token read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oidc token endpoint status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("oidc token decode: %w", err)
	}
	if tokenResp.IDToken == "" {
		return nil, fmt.Errorf("oidc token response: missing id_token")
	}
	return &tokenResp, nil
}
