package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/semver/v3"
)

func TestFindMigrationsOrdersAndFilters(t *testing.T) {
	dir := t.TempDir()

	files := []string{
		"v1.0.0.sql",
		"v1.0.1.sql",
		"v1.1.0.sql",
		"v2.0.0.sql",
		"not-a-version.sql",
	}
	for _, name := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("-- noop"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	manager := &Manager{MigrationsPath: dir}
	fromVersion, err := semver.NewVersion("1.0.0")
	if err != nil {
		t.Fatalf("fromVersion: %v", err)
	}
	toVersion, err := semver.NewVersion("1.1.0")
	if err != nil {
		t.Fatalf("toVersion: %v", err)
	}

	migrations, err := manager.findMigrations(fromVersion, toVersion)
	if err != nil {
		t.Fatalf("findMigrations: %v", err)
	}

	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migrations))
	}
	if migrations[0].Version.String() != "1.0.1" {
		t.Fatalf("expected first migration 1.0.1, got %s", migrations[0].Version.String())
	}
	if migrations[1].Version.String() != "1.1.0" {
		t.Fatalf("expected second migration 1.1.0, got %s", migrations[1].Version.String())
	}
}

func TestHasExplicitTransactionStatements(t *testing.T) {
	tests := []struct {
		name     string
		sqlText  string
		expected bool
	}{
		{
			name: "ignores comments and strings",
			sqlText: strings.Join([]string{
				"-- BEGIN",
				"/* COMMIT */",
				"INSERT INTO t (note) VALUES ('ROLLBACK');",
				"SELECT $$BEGIN$$;",
			}, "\n"),
			expected: false,
		},
		{
			name: "detects transaction statement",
			sqlText: strings.Join([]string{
				"BEGIN;",
				"UPDATE t SET x = 1;",
				"COMMIT;",
			}, "\n"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasExplicitTransactionStatements(tt.sqlText)
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestMigrateDatabaseNoMigrationsUpdatesVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mock.ExpectQuery("SELECT value FROM application WHERE key = 'version'").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("1.0.0"))
	mock.ExpectExec("UPDATE application SET value = \\$1 WHERE key = 'version'").
		WithArgs("1.1.0").
		WillReturnResult(sqlmock.NewResult(1, 1))

	manager := &Manager{
		DB:             db,
		MigrationsPath: t.TempDir(),
		AppVersion:     "1.1.0",
	}

	if err := manager.MigrateDatabase(); err != nil {
		t.Fatalf("MigrateDatabase: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
