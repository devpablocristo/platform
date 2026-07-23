# platform/sdks/clerk

Reusable Clerk Backend API clients and verification adapters.

The Go module under `go/` is intentionally provider-specific but product-agnostic.
It knows Clerk resources such as users, organizations, memberships and
invitations; it does not know consumer-specific tenancy rules.

## Go

`New` creates the Backend API client. `NewSessionVerifier` creates a
fail-closed session-token verifier based on Clerk's official Go SDK:

```go
verifier, err := clerk.NewSessionVerifier(clerk.SessionVerifierConfig{
    SecretKey:        os.Getenv("CLERK_SECRET_KEY"),
    Issuer:           "https://example.clerk.accounts.dev",
    Audience:         "my-api",
    AuthorizedParties: []string{"https://app.example.com"},
    ClockSkew:        30 * time.Second,
})
claims, err := verifier.VerifySession(ctx, token)
```

The verifier requires an exact issuer, audience and authorized-party match,
valid time claims, a subject, session ID and active organization. Pending
sessions are rejected. Applications remain responsible for resolving the
verified Clerk organization and user into their own local membership model.
For identity-scoped operations before organization selection,
`VerifyIdentity` applies the same token and session checks while allowing the
organization claims to be absent; partial organization claims remain invalid.
When remote JWKS retrieval fails because of a transport error, timeout, HTTP
`429` or HTTP `5xx`, verification returns `ErrProviderUnavailable`. Consumers
can distinguish that retryable provider failure with `errors.Is`; malformed
tokens, invalid signatures or claims, and a `kid` absent from a valid JWKS
remain `ErrInvalidSessionToken`.

The Backend API client also provides provider-scoped organization invitation,
membership and session operations:

- `ListOrgInvitations`, `GetOrgInvitation`, `RevokeOrgInvitation`
- `ListOrganizationMemberships`, `GetOrgMembership`,
  `RevokeOrgMembership`
- `ListSessions`, `GetSession`, `RevokeSession`

Revocation methods are idempotent when Clerk returns `404`. Rate-limited
responses can be inspected with `IsRateLimited` and `RetryAfter`.

`NewWebhookVerifier` verifies Clerk/Svix signatures over the original request
body and returns a typed `WebhookEvent`. User, organization, membership,
invitation and session event families decode to the corresponding resource
types. Unknown event families remain available as verified `RawWebhookData`.
The SDK does not register routes, persist events or decide how a product
reconciles provider state.
