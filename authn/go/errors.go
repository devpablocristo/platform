package authn

import "errors"

var (
	// ErrNoValidCredential indica que no hubo JWT ni API key válidos tras intentar el flujo inbound.
	ErrNoValidCredential = errors.New("authn: no valid credential")

	// ErrWrongCredentialKind indica que el Authenticator recibió un Credential de otro Kind.
	ErrWrongCredentialKind = errors.New("authn: wrong credential kind")

	// ErrProviderUnavailable indica que NO SE PUDO verificar la credencial porque el
	// proveedor de identidad no respondió.
	//
	// No significa que la credencial sea inválida, y la distinción no es cosmética: un
	// consumidor que colapse las dos cosas en 401 le miente al cliente —lo manda a
	// loguearse de nuevo cuando su token está perfecto— y además se queda sin señal,
	// porque un 401 no es un error de nadie y ninguna alarma lo mira.
	//
	// Lo envuelven: timeout, DNS o conexión rechazada contra el JWKS o el documento de
	// discovery; cualquier status fuera de 2xx; un cuerpo que no decodifica; un documento
	// sin claves usables; y un documento de discovery incompleto.
	//
	// NO lo envuelven: un `kid` que no está DESPUÉS de un refresh exitoso —eso es un token
	// firmado por una clave que este proveedor no publica—, una firma inválida, un token
	// expirado, un `alg` no permitido ni un token vacío.
	ErrProviderUnavailable = errors.New("authn: identity provider unavailable")
)
