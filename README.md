## JW6 Go Utils

A collection of helpful utilities
 - Logging utilities
 - PostgreSQL Database manager with a version based migration system


## Database Manager Usage (PostgreSQL only)

```go
package main

import (
	"log"

	jw6_utils "github.com/jw6ventures/jw6-go-utils"
	"github.com/jw6ventures/jw6-go-utils/database"
)

func main() {
	utils := &jw6_utils.Utils{LogLevel: jw6_utils.Info}

	manager := database.NewManager(database.Config{
		Driver:           "postgres",
		ConnString:       "postgres://user:pass@localhost:5432/app?sslmode=disable",
		MigrationsPath:   "migrations",
		AppVersion:       "1.2.3",
		SchemaPath:       "db.sql",
		SchemaCheckTable: "institution_links",
		Logger:           utils,
	})

	if err := manager.Initialize(); err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Use manager.DB for queries, or manager.MigrationManager for migration helpers.
}
```

Notes:
- The database manager currently supports PostgreSQL only.
- The migration manager uses an `application` table with `key`/`value` columns and it generated immediately after opening the DB connection the first time. It stores the current schema version under `key = 'version'`.
- Ensure your baseline schema (e.g., `db.sql`) inserts the starting version row into the `application` table.
  ```
	INSERT INTO application (key,value) VALUES ('version', 'v0.0.1')
  ```
- Migrations live in the directory you pass via `MigrationsPath`, with files named `vX.Y.Z.sql` (e.g., `v1.2.3.sql`). Pre-release tags are supported (e.g., `v0.1.0-rc1.sql` or `v0.1.0-RC2.sql`).
- The manager runs migrations in semantic version order from the stored DB version up to `AppVersion`.
- If a migration file includes explicit transaction statements (BEGIN/COMMIT/etc.), it is executed as-is; otherwise it runs inside a managed transaction.

## Logging Setup Example

```go
package main

import (
	jw6_utils "github.com/jw6ventures/jw6-go-utils"
)

func main() {
	// Only logs Info/Warn/Error/Fatal from helpers that accept a Logger.
	utils := &jw6_utils.Utils{LogLevel: jw6_utils.Info}

	utils.Log("Bootstrap", "main", jw6_utils.Info, "Logging ready")
}
```

Notes:
- `Utils.LogLevel` is the minimum threshold; lower levels are skipped.
- `Fatal` prints a banner but does not exit the process.
- Logs print to stdout and include ANSI color codes.
