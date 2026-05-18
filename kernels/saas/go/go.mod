module github.com/devpablocristo/platform/kernels/saas/go

go 1.26.1

require (
	github.com/devpablocristo/platform/authn/go v0.0.0-00010101000000-000000000000
	github.com/devpablocristo/platform/authz/go v0.0.0-00010101000000-000000000000
	github.com/devpablocristo/platform/errors/go v0.0.0-00010101000000-000000000000
	github.com/devpablocristo/platform/http/go v0.0.0-00010101000000-000000000000
	github.com/devpablocristo/platform/observability/go v0.0.0-00010101000000-000000000000
	github.com/devpablocristo/platform/security/go v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/stripe/stripe-go/v81 v81.4.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)








replace github.com/devpablocristo/platform/authn/go => ../../../authn/go
replace github.com/devpablocristo/platform/authz/go => ../../../authz/go
replace github.com/devpablocristo/platform/errors/go => ../../../errors/go
replace github.com/devpablocristo/platform/http/go => ../../../http/go
replace github.com/devpablocristo/platform/observability/go => ../../../observability/go
replace github.com/devpablocristo/platform/security/go => ../../../security/go
