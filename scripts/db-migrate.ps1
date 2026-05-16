$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Resolve-Path (Join-Path $scriptDir "..")
$migrationsDir = Join-Path $repoRoot "backend\\migrations"
$containerName = "aethernet-postgres"
$dbUser = "aether"
$dbName = "aethernet"

if (-not (Test-Path $migrationsDir)) {
  throw "Migrations directory not found: $migrationsDir"
}

$containerExists = docker ps --format "{{.Names}}" | Select-String -SimpleMatch $containerName
if (-not $containerExists) {
  throw "Postgres container '$containerName' is not running. Start it first with: pnpm db:up"
}

function Invoke-PsqlCommand {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Sql
  )

  $Sql | docker exec -i $containerName psql -v ON_ERROR_STOP=1 -U $dbUser -d $dbName
}

function Invoke-PsqlFile {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Sql
  )

  $Sql | docker exec -i $containerName psql -q -v ON_ERROR_STOP=1 -U $dbUser -d $dbName
}

Invoke-PsqlCommand @"
CREATE TABLE IF NOT EXISTS schema_migrations (
  version BIGINT PRIMARY KEY,
  dirty BOOLEAN NOT NULL DEFAULT FALSE,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
"@

$appliedVersions = @{}
$appliedOutput = docker exec -i $containerName psql -t -A -U $dbUser -d $dbName -c "SELECT version FROM schema_migrations ORDER BY version;"
foreach ($line in ($appliedOutput -split "`r?`n")) {
  $version = $line.Trim()
  if ($version -ne "") {
    $appliedVersions[$version] = $true
  }
}

$migrationFiles = Get-ChildItem $migrationsDir -Filter "*.up.sql" | Sort-Object Name
if ($migrationFiles.Count -eq 0) {
  Write-Host "No up migrations found in $migrationsDir"
  exit 0
}

foreach ($file in $migrationFiles) {
  $versionMatch = [regex]::Match($file.BaseName, '^\d+')
  if (-not $versionMatch.Success) {
    throw "Migration file name must start with a numeric version: $($file.Name)"
  }

  $version = $versionMatch.Value
  if ($appliedVersions.ContainsKey($version)) {
    Write-Host "Skipping migration $version ($($file.Name))"
    continue
  }

  Write-Host "Applying migration $version ($($file.Name))"
  $body = Get-Content -Raw $file.FullName
  $sql = @"
BEGIN;
$body
INSERT INTO schema_migrations(version, dirty) VALUES ($version, FALSE);
COMMIT;
"@

  Invoke-PsqlFile $sql
}

Write-Host "Database migrations complete."
