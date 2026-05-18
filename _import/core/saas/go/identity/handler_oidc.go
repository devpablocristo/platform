package identity

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	oidc "github.com/devpablocristo/core/authn/go/oidc"
	"github.com/devpablocristo/core/http/go/httperr"
	identitydto "github.com/devpablocristo/core/saas/go/identity/handler/dto"
	identitydomain "github.com/devpablocristo/core/saas/go/identity/usecases/domain"
)

type OIDCConfig struct {
	Enabled      bool
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type DiscoveryPort interface {
	VerifyToken(context.Context, string) (map[string]any, error)
}

type TokenExchangerPort interface {
	AuthorizationURL(context.Context, string, string, oidc.PKCEParams, []string) (string, error)
	ExchangeCode(context.Context, string, string) (*oidc.TokenResponse, error)
}

type principalResolver interface {
	ResolvePrincipal(context.Context, string) (identitydomain.Principal, error)
}

const maxPendingFlows = 10000

type OIDCHandler struct {
	cfg       OIDCConfig
	discovery DiscoveryPort
	exchanger TokenExchangerPort
	idSvc     principalResolver
	logger    *slog.Logger

	mu           sync.Mutex
	pendingFlows map[string]*oidcFlowState
}

type oidcFlowState struct {
	CodeVerifier string
	Nonce        string
	CreatedAt    time.Time
}

func NewOIDCHandler(cfg OIDCConfig, discovery DiscoveryPort, exchanger TokenExchangerPort, idSvc principalResolver, logger *slog.Logger) *OIDCHandler {
	if logger == nil {
		logger = slog.Default()
	}
	handler := &OIDCHandler{
		cfg:          cfg,
		discovery:    discovery,
		exchanger:    exchanger,
		idSvc:        idSvc,
		logger:       logger,
		pendingFlows: make(map[string]*oidcFlowState),
	}
	go handler.cleanupLoop()
	return handler
}

func (h *OIDCHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/oidc/config", h.configStatus)
	if !h.cfg.Enabled {
		return
	}
	mux.HandleFunc("GET /auth/oidc/authorize", h.authorize)
	mux.HandleFunc("GET /auth/oidc/callback", h.callback)
}

func (h *OIDCHandler) configStatus(w http.ResponseWriter, _ *http.Request) {
	resp := identitydto.OIDCConfigResponse{OIDCEnabled: h.cfg.Enabled}
	if h.cfg.Enabled {
		resp.IssuerURL = h.cfg.IssuerURL
		resp.Scopes = append([]string(nil), h.cfg.Scopes...)
	}
	httperr.WriteJSON(w, http.StatusOK, resp)
}

func (h *OIDCHandler) authorize(w http.ResponseWriter, r *http.Request) {
	pkce, err := oidc.GeneratePKCE()
	if err != nil {
		httperr.Write(w, http.StatusInternalServerError, httperr.CodeInternal, "failed to generate PKCE parameters")
		return
	}
	state, err := oidc.GenerateState()
	if err != nil {
		httperr.Write(w, http.StatusInternalServerError, httperr.CodeInternal, "failed to generate state")
		return
	}
	nonce, err := oidc.GenerateNonce()
	if err != nil {
		httperr.Write(w, http.StatusInternalServerError, httperr.CodeInternal, "failed to generate nonce")
		return
	}

	authURL, err := h.exchanger.AuthorizationURL(r.Context(), state, nonce, pkce, h.cfg.Scopes)
	if err != nil {
		httperr.Write(w, http.StatusBadGateway, httperr.CodeInternal, "failed to build authorization URL")
		return
	}

	h.mu.Lock()
	if len(h.pendingFlows) >= maxPendingFlows {
		h.mu.Unlock()
		httperr.Write(w, http.StatusServiceUnavailable, httperr.CodeInternal, "too many pending login flows, try again later")
		return
	}
	h.pendingFlows[state] = &oidcFlowState{
		CodeVerifier: pkce.CodeVerifier,
		Nonce:        nonce,
		CreatedAt:    time.Now().UTC(),
	}
	h.mu.Unlock()

	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *OIDCHandler) callback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errCode := q.Get("error"); errCode != "" {
		errDesc := q.Get("error_description")
		if errDesc == "" {
			errDesc = "unknown error"
		}
		httperr.Write(w, http.StatusBadRequest, "OIDC_ERROR", errCode+": "+errDesc)
		return
	}

	state := q.Get("state")
	code := q.Get("code")
	if state == "" || code == "" {
		httperr.BadRequest(w, "missing state or code parameter")
		return
	}

	h.mu.Lock()
	flow, ok := h.pendingFlows[state]
	if ok {
		delete(h.pendingFlows, state)
	}
	h.mu.Unlock()
	if !ok {
		httperr.BadRequest(w, "invalid or expired state parameter")
		return
	}
	if time.Since(flow.CreatedAt) > 10*time.Minute {
		httperr.BadRequest(w, "authorization flow expired")
		return
	}

	tokenResp, err := h.exchanger.ExchangeCode(r.Context(), code, flow.CodeVerifier)
	if err != nil {
		h.logger.Error("oidc token exchange failed", "error", err)
		httperr.Write(w, http.StatusBadGateway, "OIDC_ERROR", "token exchange failed")
		return
	}

	claims, err := h.discovery.VerifyToken(r.Context(), tokenResp.IDToken)
	if err != nil {
		h.logger.Error("oidc id token verification failed", "error", err)
		httperr.Write(w, http.StatusUnauthorized, httperr.CodeUnauthorized, "id token verification failed")
		return
	}

	if nonceClaim, _ := claims["nonce"].(string); nonceClaim != flow.Nonce {
		httperr.Write(w, http.StatusUnauthorized, httperr.CodeUnauthorized, "id token nonce mismatch")
		return
	}

	principal, err := h.idSvc.ResolvePrincipal(r.Context(), tokenResp.IDToken)
	if err != nil {
		h.logger.Warn("oidc principal resolution failed", "error", err)
		httperr.WriteJSON(w, http.StatusOK, identitydto.OIDCCallbackWarningResponse{
			AuthMethod:  "oidc",
			IDToken:     tokenResp.IDToken,
			AccessToken: tokenResp.AccessToken,
			ExpiresIn:   tokenResp.ExpiresIn,
			Claims:      claims,
			Warning:     "principal resolution failed, check claim mapping",
		})
		return
	}

	httperr.WriteJSON(w, http.StatusOK, identitydto.OIDCCallbackResponse{
		AuthMethod:  "oidc",
		IDToken:     tokenResp.IDToken,
		AccessToken: tokenResp.AccessToken,
		ExpiresIn:   tokenResp.ExpiresIn,
		TenantID:    principal.TenantID,
		Actor:       principal.Actor,
		Role:        principal.Role,
		Scopes:      append([]string(nil), principal.Scopes...),
	})
}

func (h *OIDCHandler) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.Lock()
		now := time.Now().UTC()
		for key, value := range h.pendingFlows {
			if now.Sub(value.CreatedAt) > 15*time.Minute {
				delete(h.pendingFlows, key)
			}
		}
		h.mu.Unlock()
	}
}
