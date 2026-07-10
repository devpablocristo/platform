package firebase

import (
	"strings"

	"github.com/devpablocristo/platform/authn/go/jwks"
	identity "github.com/devpablocristo/platform/kernels/saas/go/identity"
)

// JWKSURL es el endpoint JWK (formato n/e) de Google securetoken que firma los ID tokens de
// Identity Platform / Firebase. NO usar el endpoint x509/PEM: authn/go/jwks espera JWK n/e.
const JWKSURL = "https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com"

// IssuerFor devuelve el issuer canónico de los ID tokens de un proyecto de Identity Platform.
func IssuerFor(projectID string) string {
	return "https://securetoken.google.com/" + strings.TrimSpace(projectID)
}

// NewVerifier arma el TokenVerifierPort (authn/go/jwks) para Identity Platform. El verifier valida
// firma RS256 + exp contra las claves de securetoken; el issuer/audience los enforza ClaimsConfig.
func NewVerifier() *jwks.Verifier {
	return jwks.NewVerifier(JWKSURL)
}

// ClaimsConfig arma el ClaimsConfig del kernel para un proyecto de Identity Platform (issuer/aud
// derivados del projectID). Sirve para el path claim-based (deployments con custom claims de org).
// Para tokens sin org claim, usar el resolver-por-membership (identity.NewMembershipResolver).
func ClaimsConfig(projectID string) identity.ClaimsConfig {
	projectID = strings.TrimSpace(projectID)
	return identity.ClaimsConfig{
		Issuer:      IssuerFor(projectID),
		Audience:    projectID,
		OrgClaim:    "org_id",
		RoleClaim:   "role",
		ScopesClaim: "scopes",
		ActorClaim:  "sub",
	}
}
