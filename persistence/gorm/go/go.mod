module github.com/devpablocristo/platform/persistence/gorm/go

go 1.26.5

require (
	github.com/devpablocristo/platform/errors/go v0.0.0-00010101000000-000000000000
	github.com/devpablocristo/platform/security/go v0.0.0-00010101000000-000000000000
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	golang.org/x/text v0.20.0 // indirect
)

replace (
	github.com/devpablocristo/platform/errors/go => ../../../errors/go
	github.com/devpablocristo/platform/http/go => ../../../http/go
	github.com/devpablocristo/platform/security/go => ../../../security/go
)
