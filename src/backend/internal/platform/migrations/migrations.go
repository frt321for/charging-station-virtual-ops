package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"time"

	"charging-ops/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
)

//go:embed files/*.sql
var migrationFiles embed.FS

var migrationFilePattern = regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)

// Runner applies embedded SQL migrations.
type Runner struct {
	database *database.Client
}

// Migration describes a paired up/down SQL migration.
type Migration struct {
	Version string
	Name    string
	UpSQL   string
	DownSQL string
}

// Status describes whether a migration has been applied.
type Status struct {
	Version   string
	Name      string
	Applied   bool
	AppliedAt *time.Time
}

// NewRunner creates a migration runner for the connected database.
func NewRunner(database *database.Client) *Runner {
	return &Runner{database: database}
}

// Up applies all pending migrations in ascending version order.
func (r *Runner) Up(ctx context.Context) ([]string, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return nil, err
	}

	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}

	applied, err := r.applied(ctx)
	if err != nil {
		return nil, err
	}

	changed := make([]string, 0)
	for _, migration := range migrations {
		if _, ok := applied[migration.Version]; ok {
			continue
		}
		if err := r.applyUp(ctx, migration); err != nil {
			return changed, err
		}
		changed = append(changed, migration.Version+"_"+migration.Name)
	}

	return changed, nil
}

// Down rolls back the latest applied migration.
func (r *Runner) Down(ctx context.Context) ([]string, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return nil, err
	}

	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}

	latest, err := r.latestApplied(ctx)
	if err != nil {
		return nil, err
	}
	if latest == "" {
		return []string{}, nil
	}

	for i := len(migrations) - 1; i >= 0; i-- {
		if migrations[i].Version == latest {
			if err := r.applyDown(ctx, migrations[i]); err != nil {
				return nil, err
			}
			return []string{migrations[i].Version + "_" + migrations[i].Name}, nil
		}
	}

	return nil, fmt.Errorf("missing migration file for applied version %s", latest)
}

// Status returns all known migrations and their database state.
func (r *Runner) Status(ctx context.Context) ([]Status, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return nil, err
	}

	if err := r.ensureSchema(ctx); err != nil {
		return nil, err
	}

	applied, err := r.applied(ctx)
	if err != nil {
		return nil, err
	}

	statuses := make([]Status, 0, len(migrations))
	for _, migration := range migrations {
		appliedAt, ok := applied[migration.Version]
		status := Status{Version: migration.Version, Name: migration.Name, Applied: ok}
		if ok {
			status.AppliedAt = &appliedAt
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

func (r *Runner) ensureSchema(ctx context.Context) error {
	_, err := r.database.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			checksum TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure schema migrations table: %w", err)
	}
	return nil
}

func (r *Runner) applied(ctx context.Context) (map[string]time.Time, error) {
	rows, err := r.database.Pool().Query(ctx, "SELECT version, applied_at FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	result := make(map[string]time.Time)
	for rows.Next() {
		var version string
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		result[version] = appliedAt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}

	return result, nil
}

func (r *Runner) latestApplied(ctx context.Context) (string, error) {
	var version string
	err := r.database.Pool().QueryRow(ctx, `
		SELECT version
		FROM schema_migrations
		ORDER BY version DESC
		LIMIT 1
	`).Scan(&version)
	if err == nil {
		return version, nil
	}
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return "", fmt.Errorf("query latest applied migration: %w", err)
}

func (r *Runner) applyUp(ctx context.Context, migration Migration) error {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.Version, err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, migration.UpSQL); err != nil {
		return fmt.Errorf("apply migration %s: %w", migration.Version, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO schema_migrations (version, name, checksum)
		VALUES ($1, $2, $3)
	`, migration.Version, migration.Name, checksum(migration.UpSQL)); err != nil {
		return fmt.Errorf("record migration %s: %w", migration.Version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", migration.Version, err)
	}
	return nil
}

func (r *Runner) applyDown(ctx context.Context, migration Migration) error {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rollback %s: %w", migration.Version, err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, migration.DownSQL); err != nil {
		return fmt.Errorf("rollback migration %s: %w", migration.Version, err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", migration.Version); err != nil {
		return fmt.Errorf("remove migration record %s: %w", migration.Version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit rollback %s: %w", migration.Version, err)
	}
	return nil
}

func loadMigrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "files")
	if err != nil {
		return nil, fmt.Errorf("read migration files: %w", err)
	}

	grouped := make(map[string]*Migration)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		version, name, direction := matches[1], matches[2], matches[3]
		migration := grouped[version]
		if migration == nil {
			migration = &Migration{Version: version, Name: name}
			grouped[version] = migration
		}

		content, err := migrationFiles.ReadFile("files/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if direction == "up" {
			migration.UpSQL = string(content)
		} else {
			migration.DownSQL = string(content)
		}
	}

	migrations := make([]Migration, 0, len(grouped))
	for _, migration := range grouped {
		if migration.UpSQL == "" || migration.DownSQL == "" {
			return nil, fmt.Errorf("migration %s_%s must have up and down SQL", migration.Version, migration.Name)
		}
		migrations = append(migrations, *migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func checksum(sql string) string {
	sum := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(sum[:])
}
