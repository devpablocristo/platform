module github.com/devpablocristo/platform/security/go

go 1.26.5

require (
	github.com/devpablocristo/platform/errors/go v0.0.0-00010101000000-000000000000
	github.com/devpablocristo/platform/http/go v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
)

replace (
	github.com/devpablocristo/platform/errors/go => ../../errors/go
	github.com/devpablocristo/platform/http/go => ../../http/go
)
