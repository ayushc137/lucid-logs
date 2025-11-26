package model

import "github.com/surrealdb/surrealdb.go"

// DBAuth aliases the driver type so handlers don't need to import surrealdb directly.
type DBAuth = surrealdb.Auth
