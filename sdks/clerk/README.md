# platform/sdks/clerk

Reusable Clerk Backend API clients.

The Go module under `go/` is intentionally provider-specific but product-agnostic.
It knows Clerk resources such as users, organizations, memberships and
invitations; it does not know consumer-specific tenancy rules.
