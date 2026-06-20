package main

import "embed"

// MigrationsFS embeds all migration files for sales-service.
// The "migrations" subdirectory name is required by the go-shared migrate helper
// (iofs.New expects the files under a named subdirectory of the provided FS).
//
// El entrypoint de sales-service es el main.go de la raíz (go build -o sales-service .),
// que comparte paquete (main) con este archivo, por lo que main.go referencia
// MigrationsFS directamente sin import.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
