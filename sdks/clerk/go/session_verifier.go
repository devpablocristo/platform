package clerk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	clerksdk "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
)

const defaultJWKCacheTTL = time.Hour

var (
	ErrInvalidSessionToken  = errors.New("clerk: invalid session token")
	ErrOrganizationRequired = errors.New("clerk: active organization required")
	ErrPendingSession       = errors.New("clerk: session is pending")
)

type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusPending SessionStatus = "pending"
)

// SessionVerifierConfig configures fail-closed Clerk session-token
// verification. Issuer, Audience and AuthorizedParties are all required.
//
// SecretKey is used to retrieve Clerk's JWKS. PublicKeyPEM may be supplied
// instead for networkless verification; when both are present, PublicKeyPEM
// takes precedence.
type SessionVerifierConfig struct {
	SecretKey         string
	BaseURL           string
	Client            *http.Client
	PublicKeyPEM      string
	Issuer            string
	Audience          string
	AuthorizedParties []string
	ClockSkew         time.Duration
	Clock             clerksdk.Clock
}

// SessionClaims is the normalized, verified subset of a Clerk session token.
// Provider-specific values intentionally remain in this SDK so consumers can
// map them explicitly into their own identity contracts.
type SessionClaims struct {
	Subject                 string
	SessionID               string
	OrganizationID          string
	OrganizationSlug        string
	OrganizationRole        string
	OrganizationPermissions []string
	Status                  SessionStatus
	Issuer                  string
	Audience                []string
	AuthorizedParty         string
	IssuedAt                time.Time
	NotBefore               time.Time
	ExpiresAt               time.Time
}

type SessionVerifier struct {
	issuer            string
	audience          string
	authorizedParties map[string]struct{}
	clockSkew         time.Duration
	clock             clerksdk.Clock
	staticJWK         *clerksdk.JSONWebKey
	jwksClient        *jwks.Client

	cacheMu sync.Mutex
	cache   map[string]cachedJWK
}

type cachedJWK struct {
	key       *clerksdk.JSONWebKey
	expiresAt time.Time
}

type sessionStatusClaims struct {
	Status string `json:"sts"`
}

func NewSessionVerifier(config SessionVerifierConfig) (*SessionVerifier, error) {
	issuer := strings.TrimSpace(config.Issuer)
	if issuer == "" {
		return nil, fmt.Errorf("clerk: issuer is required")
	}
	audience := strings.TrimSpace(config.Audience)
	if audience == "" {
		return nil, fmt.Errorf("clerk: audience is required")
	}
	if config.ClockSkew < 0 {
		return nil, fmt.Errorf("clerk: clock skew cannot be negative")
	}

	authorizedParties := make(map[string]struct{}, len(config.AuthorizedParties))
	for _, party := range config.AuthorizedParties {
		party = strings.TrimSpace(party)
		if party == "" {
			continue
		}
		authorizedParties[party] = struct{}{}
	}
	if len(authorizedParties) == 0 {
		return nil, fmt.Errorf("clerk: at least one authorized party is required")
	}

	clock := config.Clock
	if clock == nil {
		clock = clerksdk.NewClock()
	}

	verifier := &SessionVerifier{
		issuer:            issuer,
		audience:          audience,
		authorizedParties: authorizedParties,
		clockSkew:         config.ClockSkew,
		clock:             clock,
		cache:             make(map[string]cachedJWK),
	}

	if publicKey := strings.TrimSpace(config.PublicKeyPEM); publicKey != "" {
		key, err := clerksdk.JSONWebKeyFromPEM(publicKey)
		if err != nil {
			return nil, fmt.Errorf("clerk: parse public key: %w", err)
		}
		verifier.staticJWK = key
		return verifier, nil
	}

	secretKey := strings.TrimSpace(config.SecretKey)
	if secretKey == "" {
		return nil, fmt.Errorf("clerk: secret key or public key is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	httpClient := config.Client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	verifier.jwksClient = jwks.NewClient(&clerksdk.ClientConfig{
		BackendConfig: clerksdk.BackendConfig{
			HTTPClient: httpClient,
			URL:        clerksdk.String(baseURL),
			Key:        clerksdk.String(secretKey),
		},
	})
	return verifier, nil
}

func (v *SessionVerifier) VerifySession(ctx context.Context, token string) (SessionClaims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return SessionClaims{}, fmt.Errorf("%w: token is empty", ErrInvalidSessionToken)
	}

	key, err := v.keyForToken(ctx, token)
	if err != nil {
		return SessionClaims{}, fmt.Errorf("%w: %v", ErrInvalidSessionToken, err)
	}

	statusClaims := &sessionStatusClaims{}
	claims, err := clerkjwt.Verify(ctx, &clerkjwt.VerifyParams{
		Token: token,
		JWK:   key,
		Clock: v.clock,
		// Passing the exact configured issuer as Clerk's proxy URL makes the
		// official verifier support custom Clerk domains while preserving an
		// exact (rather than suffix-based) issuer check.
		ProxyURL: clerksdk.String(v.issuer),
		CustomClaimsConstructor: func(context.Context) any {
			return statusClaims
		},
		Leeway: v.clockSkew,
		AuthorizedPartyHandler: func(party string) bool {
			_, ok := v.authorizedParties[party]
			return ok
		},
	})
	if err != nil {
		return SessionClaims{}, fmt.Errorf("%w: %v", ErrInvalidSessionToken, err)
	}

	if claims.Issuer != v.issuer {
		return SessionClaims{}, fmt.Errorf("%w: issuer does not match", ErrInvalidSessionToken)
	}
	if !slices.Contains(claims.Audience, v.audience) {
		return SessionClaims{}, fmt.Errorf("%w: audience does not match", ErrInvalidSessionToken)
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return SessionClaims{}, fmt.Errorf("%w: subject is missing", ErrInvalidSessionToken)
	}
	if strings.TrimSpace(claims.SessionID) == "" {
		return SessionClaims{}, fmt.Errorf("%w: session id is missing", ErrInvalidSessionToken)
	}
	if claims.IssuedAt == nil || claims.NotBefore == nil || claims.Expiry == nil {
		return SessionClaims{}, fmt.Errorf("%w: required time claims are missing", ErrInvalidSessionToken)
	}
	if strings.TrimSpace(claims.ActiveOrganizationID) == "" {
		return SessionClaims{}, ErrOrganizationRequired
	}
	if strings.TrimSpace(claims.ActiveOrganizationRole) == "" {
		return SessionClaims{}, fmt.Errorf("%w: organization role is missing", ErrInvalidSessionToken)
	}

	issuedAt := time.Unix(*claims.IssuedAt, 0).UTC()
	if issuedAt.After(v.clock.Now().UTC().Add(v.clockSkew)) {
		return SessionClaims{}, fmt.Errorf("%w: issued-at time is in the future", ErrInvalidSessionToken)
	}

	status := SessionStatus(strings.TrimSpace(statusClaims.Status))
	if status == "" {
		status = SessionStatusActive
	}
	if status == SessionStatusPending {
		return SessionClaims{}, ErrPendingSession
	}
	if status != SessionStatusActive {
		return SessionClaims{}, fmt.Errorf("%w: unsupported session status", ErrInvalidSessionToken)
	}

	return SessionClaims{
		Subject:                 strings.TrimSpace(claims.Subject),
		SessionID:               strings.TrimSpace(claims.SessionID),
		OrganizationID:          strings.TrimSpace(claims.ActiveOrganizationID),
		OrganizationSlug:        strings.TrimSpace(claims.ActiveOrganizationSlug),
		OrganizationRole:        strings.TrimSpace(claims.ActiveOrganizationRole),
		OrganizationPermissions: append([]string(nil), claims.ActiveOrganizationPermissions...),
		Status:                  status,
		Issuer:                  claims.Issuer,
		Audience:                append([]string(nil), claims.Audience...),
		AuthorizedParty:         strings.TrimSpace(claims.AuthorizedParty),
		IssuedAt:                issuedAt,
		NotBefore:               time.Unix(*claims.NotBefore, 0).UTC(),
		ExpiresAt:               time.Unix(*claims.Expiry, 0).UTC(),
	}, nil
}

func (v *SessionVerifier) keyForToken(ctx context.Context, token string) (*clerksdk.JSONWebKey, error) {
	if v.staticJWK != nil {
		return v.staticJWK, nil
	}

	decoded, err := clerkjwt.Decode(ctx, &clerkjwt.DecodeParams{Token: token})
	if err != nil {
		return nil, err
	}
	keyID := strings.TrimSpace(decoded.KeyID)
	if keyID == "" {
		return nil, fmt.Errorf("missing jwt kid header claim")
	}

	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()

	now := v.clock.Now().UTC()
	if entry, ok := v.cache[keyID]; ok && now.Before(entry.expiresAt) {
		return entry.key, nil
	}
	for id, entry := range v.cache {
		if !now.Before(entry.expiresAt) {
			delete(v.cache, id)
		}
	}

	key, err := clerkjwt.GetJSONWebKey(ctx, &clerkjwt.GetJSONWebKeyParams{
		KeyID:      keyID,
		JWKSClient: v.jwksClient,
	})
	if err != nil {
		return nil, err
	}
	if key == nil || key.Algorithm != "RS256" || (key.Use != "" && key.Use != "sig") {
		return nil, fmt.Errorf("invalid json web key")
	}
	v.cache[keyID] = cachedJWK{key: key, expiresAt: now.Add(defaultJWKCacheTTL)}
	return key, nil
}
