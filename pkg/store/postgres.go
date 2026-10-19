package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/vitorecomp/go-ls/pkg/models"
)

const postgresSchema = `
CREATE TABLE IF NOT EXISTS scans (
	id TEXT PRIMARY KEY,
	tag TEXT NOT NULL DEFAULT '',
	computer_id TEXT NOT NULL DEFAULT '',
	hostname TEXT NOT NULL,
	drive_id TEXT NOT NULL DEFAULT 'default-drive',
	root_path TEXT NOT NULL,
	hash_algo TEXT NOT NULL,
	root_hash TEXT NOT NULL,
	total_files INTEGER NOT NULL,
	total_dirs INTEGER NOT NULL,
	total_bytes BIGINT NOT NULL,
	version INTEGER NOT NULL DEFAULT 1,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE scans ADD COLUMN IF NOT EXISTS computer_id TEXT NOT NULL DEFAULT '';
ALTER TABLE scans ADD COLUMN IF NOT EXISTS drive_id TEXT NOT NULL DEFAULT 'default-drive';
ALTER TABLE scans ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE scans ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE scans SET computer_id = hostname WHERE computer_id = '' AND hostname <> '';
UPDATE scans SET drive_id = 'default-drive' WHERE drive_id = '';

DELETE FROM scans a USING scans b
WHERE a.computer_id = b.computer_id
  AND a.drive_id = b.drive_id
  AND a.root_path = b.root_path
  AND (a.updated_at, a.created_at, a.id) < (b.updated_at, b.created_at, b.id);

CREATE TABLE IF NOT EXISTS scan_entries (
	id BIGSERIAL PRIMARY KEY,
	scan_id TEXT NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
	rel_path TEXT NOT NULL,
	abs_path TEXT NOT NULL,
	name TEXT NOT NULL,
	is_dir BOOLEAN NOT NULL,
	size BIGINT NOT NULL,
	mode TEXT NOT NULL,
	owner_name TEXT NOT NULL,
	group_name TEXT NOT NULL,
	mod_time TIMESTAMPTZ NOT NULL,
	hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS scan_history (
	id BIGSERIAL PRIMARY KEY,
	scan_id TEXT NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
	version INTEGER NOT NULL,
	tag TEXT NOT NULL DEFAULT '',
	previous_hash TEXT NOT NULL DEFAULT '',
	root_hash TEXT NOT NULL,
	total_files INTEGER NOT NULL,
	total_dirs INTEGER NOT NULL,
	total_bytes BIGINT NOT NULL,
	identical BOOLEAN NOT NULL DEFAULT FALSE,
	added INTEGER NOT NULL DEFAULT 0,
	deleted INTEGER NOT NULL DEFAULT 0,
	modified INTEGER NOT NULL DEFAULT 0,
	changes_json JSONB NOT NULL DEFAULT '[]'::jsonb,
	created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_scans_computer_dir ON scans(computer_id, drive_id, root_path);
CREATE UNIQUE INDEX IF NOT EXISTS idx_scans_computer_dir_unique ON scans(computer_id, drive_id, root_path);
CREATE INDEX IF NOT EXISTS idx_scan_entries_scan_id ON scan_entries(scan_id);
CREATE INDEX IF NOT EXISTS idx_scan_entries_hash ON scan_entries(hash);
CREATE INDEX IF NOT EXISTS idx_scan_history_scan_id ON scan_history(scan_id, version DESC);
`

// PostgresStore persists scans and entries in PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore connects to PostgreSQL using dbURL, retrying briefly while the
// database container initializes, and applies schema migrations.
func NewPostgresStore(ctx context.Context, dbURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("opening postgres connection: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	var pingErr error
	for attempt := 0; attempt < 20; attempt++ {
		pingErr = db.PingContext(ctx)
		if pingErr == nil {
			break
		}
		select {
		case <-ctx.Done():
			_ = db.Close()
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	if pingErr != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging postgres database: %w", pingErr)
	}

	if _, err := db.ExecContext(ctx, postgresSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrating postgres schema: %w", err)
	}
	return &PostgresStore{db: db}, nil
}

// SaveScan stores or updates a directory ScanRecord under its computer and drive inside a
// database transaction, recording the revision diff in scan_history.
func (p *PostgresStore) SaveScan(ctx context.Context, scan models.ScanRecord) (*models.ScanSummary, error) {
	probe := NormalizeScan(scan)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	lockKey := probe.ComputerID + "|" + probe.DriveID + "|" + probe.RootPath
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
		return nil, fmt.Errorf("acquiring scan lock: %w", err)
	}

	var existing *models.ScanRecord
	if scan.ID != "" {
		if rec, err := p.getScanTx(ctx, tx, scan.ID); err == nil {
			existing = rec
		} else if !errors.Is(err, ErrScanNotFound) {
			return nil, err
		}
	}
	if existing == nil {
		var matchID string
		err := tx.QueryRowContext(ctx, `
			SELECT id FROM scans
			WHERE computer_id = $1 AND drive_id = $2 AND root_path = $3
			LIMIT 1
			FOR UPDATE
		`, probe.ComputerID, probe.DriveID, probe.RootPath).Scan(&matchID)
		if err == nil && matchID != "" {
			rec, getErr := p.getScanTx(ctx, tx, matchID)
			if getErr != nil {
				return nil, getErr
			}
			existing = rec
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("looking up existing scan: %w", err)
		}
	}

	merged := MergeScanUpdate(existing, scan)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO scans (
			id, tag, computer_id, hostname, drive_id, root_path, hash_algo, root_hash,
			total_files, total_dirs, total_bytes, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			tag = EXCLUDED.tag,
			computer_id = EXCLUDED.computer_id,
			hostname = EXCLUDED.hostname,
			drive_id = EXCLUDED.drive_id,
			root_path = EXCLUDED.root_path,
			hash_algo = EXCLUDED.hash_algo,
			root_hash = EXCLUDED.root_hash,
			total_files = EXCLUDED.total_files,
			total_dirs = EXCLUDED.total_dirs,
			total_bytes = EXCLUDED.total_bytes,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
	`,
		merged.ID,
		merged.Tag,
		merged.ComputerID,
		merged.Hostname,
		merged.DriveID,
		merged.RootPath,
		merged.HashAlgo,
		merged.RootHash,
		merged.TotalFiles,
		merged.TotalDirs,
		merged.TotalBytes,
		merged.Version,
		merged.CreatedAt,
		merged.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting scan summary: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM scan_entries WHERE scan_id = $1`, merged.ID); err != nil {
		return nil, fmt.Errorf("clearing previous scan entries: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO scan_entries (
			scan_id, rel_path, abs_path, name, is_dir, size,
			mode, owner_name, group_name, mod_time, hash
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`)
	if err != nil {
		return nil, fmt.Errorf("preparing entry insert statement: %w", err)
	}
	defer stmt.Close()

	for _, e := range merged.Entries {
		if _, err := stmt.ExecContext(ctx,
			merged.ID,
			e.RelPath,
			e.AbsPath,
			e.Name,
			e.IsDir,
			e.Size,
			e.Mode,
			e.Owner,
			e.Group,
			e.ModTime,
			e.Hash,
		); err != nil {
			return nil, fmt.Errorf("inserting scan entry %q: %w", e.RelPath, err)
		}
	}

	if (existing == nil || merged.Version > existing.Version) && len(merged.History) > 0 {
		latestHist := merged.History[0]
		changes := latestHist.Changes
		if changes == nil {
			changes = []models.DiffItem{}
		}
		changesJSON, err := json.Marshal(changes)
		if err != nil || len(changesJSON) == 0 || string(changesJSON) == "null" {
			changesJSON = []byte("[]")
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO scan_history (
				scan_id, version, tag, previous_hash, root_hash,
				total_files, total_dirs, total_bytes, identical,
				added, deleted, modified, changes_json, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		`,
			merged.ID,
			latestHist.Version,
			latestHist.Tag,
			latestHist.PreviousHash,
			latestHist.RootHash,
			latestHist.TotalFiles,
			latestHist.TotalDirs,
			latestHist.TotalBytes,
			latestHist.Identical,
			latestHist.Added,
			latestHist.Deleted,
			latestHist.Modified,
			string(changesJSON),
			latestHist.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("inserting scan history entry: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing scan transaction: %w", err)
	}

	summary := merged.ScanSummary
	return &summary, nil
}

// ListScans returns all stored scan summaries ordered by UpdatedAt descending.
func (p *PostgresStore) ListScans(ctx context.Context) ([]models.ScanSummary, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, tag, computer_id, hostname, drive_id, root_path, hash_algo, root_hash,
		       total_files, total_dirs, total_bytes, version, created_at, updated_at
		FROM scans
		ORDER BY updated_at DESC, created_at DESC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying scans: %w", err)
	}
	defer rows.Close()

	var summaries []models.ScanSummary
	for rows.Next() {
		var s models.ScanSummary
		if err := rows.Scan(
			&s.ID, &s.Tag, &s.ComputerID, &s.Hostname, &s.DriveID, &s.RootPath, &s.HashAlgo, &s.RootHash,
			&s.TotalFiles, &s.TotalDirs, &s.TotalBytes, &s.Version, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning summary row: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// ListComputers returns all tracked computers and their directory snapshots.
func (p *PostgresStore) ListComputers(ctx context.Context) ([]models.Computer, error) {
	summaries, err := p.ListScans(ctx)
	if err != nil {
		return nil, err
	}
	return GroupScansByComputer(summaries), nil
}

// GetScan retrieves a full ScanRecord (summary + entries + history) by ID.
func (p *PostgresStore) GetScan(ctx context.Context, scanID string) (*models.ScanRecord, error) {
	return p.queryScan(ctx, p.db, scanID, false)
}

func (p *PostgresStore) getScanTx(ctx context.Context, tx *sql.Tx, scanID string) (*models.ScanRecord, error) {
	return p.queryScan(ctx, tx, scanID, true)
}

type sqlQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (p *PostgresStore) queryScan(ctx context.Context, q sqlQueryer, scanID string, forUpdate bool) (*models.ScanRecord, error) {
	query := `
		SELECT id, tag, computer_id, hostname, drive_id, root_path, hash_algo, root_hash,
		       total_files, total_dirs, total_bytes, version, created_at, updated_at
		FROM scans
		WHERE id = $1
	`
	if forUpdate {
		query += " FOR UPDATE"
	}

	var rec models.ScanRecord
	err := q.QueryRowContext(ctx, query, scanID).Scan(
		&rec.ID, &rec.Tag, &rec.ComputerID, &rec.Hostname, &rec.DriveID, &rec.RootPath, &rec.HashAlgo, &rec.RootHash,
		&rec.TotalFiles, &rec.TotalDirs, &rec.TotalBytes, &rec.Version, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", ErrScanNotFound, scanID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying scan %q: %w", scanID, err)
	}

	rows, err := q.QueryContext(ctx, `
		SELECT rel_path, abs_path, name, is_dir, size, mode,
		       owner_name, group_name, mod_time, hash
		FROM scan_entries
		WHERE scan_id = $1
		ORDER BY rel_path ASC
	`, scanID)
	if err != nil {
		return nil, fmt.Errorf("querying entries for scan %q: %w", scanID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var e models.ScanEntry
		if err := rows.Scan(
			&e.RelPath, &e.AbsPath, &e.Name, &e.IsDir, &e.Size, &e.Mode,
			&e.Owner, &e.Group, &e.ModTime, &e.Hash,
		); err != nil {
			return nil, fmt.Errorf("scanning entry row: %w", err)
		}
		rec.Entries = append(rec.Entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	history, err := p.queryScanHistoryRows(ctx, q, scanID)
	if err != nil {
		return nil, err
	}
	rec.History = history
	return &rec, nil
}

// GetScanHistory retrieves the historical revisions of a stored directory snapshot by ID.
func (p *PostgresStore) GetScanHistory(ctx context.Context, scanID string) ([]models.ScanHistoryEntry, error) {
	var exists string
	err := p.db.QueryRowContext(ctx, `SELECT id FROM scans WHERE id = $1`, scanID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", ErrScanNotFound, scanID)
	}
	if err != nil {
		return nil, fmt.Errorf("checking scan existence %q: %w", scanID, err)
	}

	return p.queryScanHistoryRows(ctx, p.db, scanID)
}

func (p *PostgresStore) queryScanHistoryRows(ctx context.Context, q sqlQueryer, scanID string) ([]models.ScanHistoryEntry, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT version, tag, previous_hash, root_hash, total_files, total_dirs, total_bytes,
		       identical, added, deleted, modified, changes_json, created_at
		FROM scan_history
		WHERE scan_id = $1
		ORDER BY version DESC
	`, scanID)
	if err != nil {
		return nil, fmt.Errorf("querying history for scan %q: %w", scanID, err)
	}
	defer rows.Close()

	var history []models.ScanHistoryEntry
	for rows.Next() {
		var h models.ScanHistoryEntry
		var rawChanges []byte
		if err := rows.Scan(
			&h.Version, &h.Tag, &h.PreviousHash, &h.RootHash,
			&h.TotalFiles, &h.TotalDirs, &h.TotalBytes,
			&h.Identical, &h.Added, &h.Deleted, &h.Modified,
			&rawChanges, &h.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning history row: %w", err)
		}
		if len(rawChanges) > 0 {
			if err := json.Unmarshal(rawChanges, &h.Changes); err != nil {
				return nil, fmt.Errorf("unmarshaling history changes for scan %q (version %d): %w", scanID, h.Version, err)
			}
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

// DiffScans compares two stored scans by ID.
func (p *PostgresStore) DiffScans(ctx context.Context, baseScanID, targetScanID string) (*models.DiffReport, error) {
	baseScan, err := p.GetScan(ctx, baseScanID)
	if err != nil {
		return nil, err
	}
	targetScan, err := p.GetScan(ctx, targetScanID)
	if err != nil {
		return nil, err
	}
	return ComputeDiff(*baseScan, *targetScan), nil
}

// FindDuplicates finds duplicate files in a single scan or across all stored scans.
func (p *PostgresStore) FindDuplicates(ctx context.Context, scanID string) ([]models.DuplicateGroup, error) {
	if scanID != "" {
		var exists string
		err := p.db.QueryRowContext(ctx, `SELECT id FROM scans WHERE id = $1`, scanID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrScanNotFound, scanID)
		}
		if err != nil {
			return nil, fmt.Errorf("checking scan existence %q: %w", scanID, err)
		}
	}

	rows, err := p.db.QueryContext(ctx, `
		WITH dup_keys AS (
			SELECT hash, size
			FROM scan_entries
			WHERE NOT is_dir AND hash <> '' AND size <> 0
			  AND ($1 = '' OR scan_id = $1)
			GROUP BY hash, size
			HAVING COUNT(*) >= 2
		)
		SELECT e.hash, e.size, s.id, s.computer_id, s.hostname, s.drive_id, s.root_path, e.rel_path, e.abs_path
		FROM scan_entries e
		JOIN scans s ON s.id = e.scan_id
		JOIN dup_keys d ON d.hash = e.hash AND d.size = e.size
		WHERE NOT e.is_dir AND e.hash <> '' AND e.size <> 0
		  AND ($1 = '' OR e.scan_id = $1)
		ORDER BY e.hash ASC, e.size ASC, s.id ASC, e.rel_path ASC
	`, scanID)
	if err != nil {
		return nil, fmt.Errorf("querying duplicate files: %w", err)
	}
	defer rows.Close()

	type dupKey struct {
		hash string
		size int64
	}
	groups := make(map[dupKey][]models.DuplicateFileRef)

	for rows.Next() {
		var (
			hash string
			size int64
			ref  models.DuplicateFileRef
		)
		if err := rows.Scan(
			&hash, &size,
			&ref.ScanID, &ref.ComputerID, &ref.Hostname, &ref.DriveID, &ref.RootPath,
			&ref.RelPath, &ref.AbsPath,
		); err != nil {
			return nil, fmt.Errorf("scanning duplicate file row: %w", err)
		}
		k := dupKey{hash: hash, size: size}
		groups[k] = append(groups[k], ref)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var result []models.DuplicateGroup
	for k, refs := range groups {
		if len(refs) < 2 {
			continue
		}
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].ScanID != refs[j].ScanID {
				return refs[i].ScanID < refs[j].ScanID
			}
			return refs[i].RelPath < refs[j].RelPath
		})
		result = append(result, models.DuplicateGroup{
			Hash:        k.hash,
			Size:        k.size,
			Count:       len(refs),
			WastedBytes: int64(len(refs)-1) * k.size,
			Files:       refs,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].WastedBytes != result[j].WastedBytes {
			return result[i].WastedBytes > result[j].WastedBytes
		}
		return result[i].Hash < result[j].Hash
	})

	return result, nil
}

// Close closes the underlying PostgreSQL connection pool.
func (p *PostgresStore) Close() error {
	return p.db.Close()
}
