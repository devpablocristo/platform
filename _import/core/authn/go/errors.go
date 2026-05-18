package authn

import "errors"

var (
	// ErrNoValidCredential indica que no hubo JWT ni API key válidos tras intentar el flujo inbound.
	ErrNoValidCredential = errors.New("authn: no valid credential")

	// ErrWrongCredentialKind indica que el Authenticator recibió un Credential de otro Kind.
	ErrWrongCredentialKind = errors.New("authn: wrong credential kind")
)
