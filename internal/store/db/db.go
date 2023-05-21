package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"github.com/denisschmidt/uploader/internal/store"
	"github.com/denisschmidt/uploader/internal/store/db/file"
	"github.com/denisschmidt/uploader/internal/types"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"path"
	"sort"
	"strconv"
	"time"
)

const (
	timeFormat = time.RFC3339
)

type DB struct {
	ctx       *sql.DB
	chunkSize int
}

type dbMigration struct {
	version int
	query   string
}

//go:embed migrations/*.sql
var migrationsFs embed.FS // is an embedded filesystem that contains the migration SQL files

func New(path string, defaultChunkSize int, optimizeForLiteStream bool) store.Store {
	return NewWithChunkSize(path, defaultChunkSize, optimizeForLiteStream)
}

func NewWithChunkSize(path string, chunkSize int, optimizeForLiteStream bool) *DB {
	log.Printf("reading DB from %s", path)
	ctx, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := ctx.Exec(`
		PRAGMA temp_store = FILE;
		PRAGMA journal_mode = WAL;
	`); err != nil {
		log.Fatalf("failed to set up pragmas database: %v", err)
	}

	if optimizeForLiteStream {
		if _, err := ctx.Exec(`
			PRAGMA busy_timeout = 5000;
			PRAGMA synchronous = NORMAL;
			PRAGMA wal_autocheckpoint = 0;
		`); err != nil {
			log.Fatalf("failed to set up Litestream pragmas %v", err)
		}
	}

	migrations(ctx)

	return &DB{
		ctx:       ctx,
		chunkSize: chunkSize,
	}
}

func (d DB) InsertRecord(reader io.Reader, metadata types.Metadata) error {
	log.Printf("Create a new record %s", metadata.ID)

	w := file.NewWriter(d.ctx, metadata.ID, d.chunkSize)
	// copy the content from the reader (input) to the Writer instance (w)
	if _, err := io.Copy(w, reader); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	_, err := d.ctx.Exec(`
	INSERT INTO
		records
	(
		id,
		filename,
		note,
		content_type,
		create_at
	)
	VALUES(?,?,?,?,?)`,
		metadata.ID,
		metadata.Filename,
		metadata.Note,
		metadata.ContentType,
		metadata.CreateAt.UTC().Format(timeFormat),
	)

	if err != nil {
		log.Printf("failed to insert record into `records` table: %v", err)
		return err
	}

	return nil
}

func (d DB) GetRecord(id types.ID) (types.UploadRecord, error) {
	metadata, err := d.GetMetadata(id)
	if err != nil {
		return types.UploadRecord{}, err
	}

	r, err := file.NewReader(d.ctx, id)
	if err != nil {
		return types.UploadRecord{}, err
	}

	return types.UploadRecord{
		Metadata: metadata,
		Reader:   r,
	}, nil
}

func (d DB) GetMetadata(id types.ID) (types.Metadata, error) {
	var filename string
	var note string
	var contentType string
	var createAtTime string

	err := d.ctx.QueryRow(`
		SELECT 
		    filename,
			note,
			content_type,
			create_at
		FROM
		    records
		WHERE
		    id=?`, id).Scan(&filename, &note, &contentType, &createAtTime)
	if err == sql.ErrNoRows {
		return types.Metadata{}, types.ErrFileNotExists{
			ID: id,
		}
	}
	if err != nil {
		return types.Metadata{}, err
	}

	createAt, err := time.Parse(time.RFC3339, createAtTime)
	if err != nil {
		return types.Metadata{}, err
	}

	return types.Metadata{
		ID:          id,
		Filename:    types.Filename(filename),
		Note:        types.Note(note),
		ContentType: types.ContentType(contentType),
		CreateAt:    createAt,
	}, nil
}

func (d DB) UpdateRecordMetadata(id types.ID, metadata types.Metadata) error {
	res, err := d.ctx.Exec(`
		UPDATE records
		SET
			filename = ?,
			note = ?
		WHERE
			id=?
	`, metadata.Filename, metadata.Note, id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return types.ErrFileNotExists{ID: id}
	}

	return nil
}

func (d DB) DeleteRecord(id types.ID) error {
	tx, err := d.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM
		records
	WHERE
		id=?`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM
		metadata
	WHERE
		id=?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func migrations(ctx *sql.DB) {
	var currentVersion int
	if err := ctx.QueryRow(`PRAGMA user_version`).Scan(&currentVersion); err != nil {
		log.Fatalf("failed to get user_version: %v", err)
	}

	migrations, err := getMigrationsQuery()

	if err != nil {
		log.Fatalf("error loading database migrations: %v", err)
	}

	log.Printf("start migration stats: %d/%d", currentVersion, len(migrations))

	for _, migration := range migrations {
		if migration.version <= currentVersion {
			continue
		}
		// starts a new transaction with an empty context and default transaction options
		// if any operation within the transaction fails, the whole transaction will be rolled back
		// ensuring the database remains in a consistent state.
		tx, err := ctx.BeginTx(context.Background(), nil)
		if err != nil {
			log.Fatalf("failed to create transaction %d: %v", migration.version, err)
		}

		_, err = tx.Exec(migration.query)
		if err != nil {
			log.Fatalf("failed to perform DB migration %d: %v", migration.version, err)
		}

		_, err = tx.Exec(fmt.Sprintf(`pragma user_version=%d`, migration.version))
		if err != nil {
			log.Fatalf("failed to update DB version to %d: %v", migration.version, err)
		}

		if err = tx.Commit(); err != nil {
			log.Fatalf("failed to commit migration %d: %v", migration.version, err)
		}

		log.Printf("end migration stats: %d/%d", migration.version, len(migrations))
	}
}

func getMigrationsQuery() ([]dbMigration, error) {
	migrations := []dbMigration{}
	dirname := "migrations"

	entries, err := migrationsFs.ReadDir(dirname)
	if err != nil {
		return []dbMigration{}, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version := getMigrationVersion(entry.Name())

		query, err := migrationsFs.ReadFile(path.Join(dirname, entry.Name()))
		if err != nil {
			return []dbMigration{}, err
		}

		migrations = append(migrations, dbMigration{version: version, query: string(query)})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

func getMigrationVersion(filename string) int {
	version, err := strconv.ParseInt(filename[:3], 10, 32)
	if err != nil {
		log.Fatalf("migration version is wrong: %v", filename)
	}
	return int(version)
}
