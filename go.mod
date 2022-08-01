module github.com/tm-acme-shop/acme-shop-users-service

go 1.21

require (
	github.com/tm-acme-shop/acme-shop-shared-go v0.0.0
	github.com/gin-gonic/gin v1.9.1
	github.com/lib/pq v1.10.9
	golang.org/x/crypto v0.16.0
)

replace github.com/tm-acme-shop/acme-shop-shared-go => ../acme-shop-shared-go
