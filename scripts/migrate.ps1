# Run database migrations (use before scaling API replicas or via docker compose migrate service).
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot ".." "apps" "api")
go run ./cmd/migrate
