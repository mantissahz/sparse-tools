package sparse

import (
	"database/sql"
	"fmt"

	_ "github.com/glebarez/go-sqlite"
)

const (
	dbFileSuffix = ".db"

	syncSanpshotTable  = "snapshots"
	syncDataChunkTable = "sent_chunks"
)

type SyncDB struct {
	db *sql.DB
}

func NewSyncDB(snapshotName string) (*SyncDB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("%s%s", snapshotName, dbFileSuffix))
	if err != nil {
		return nil, err
	}

	syncDB := &SyncDB{db: db}
	if err := syncDB.initialize(); err != nil {
		return nil, err
	}

	return syncDB, nil
}

func (s *SyncDB) initialize() error {
	createSnapshotTableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ctime DATETIME NOT NULL,
			mtime DATETIME NOT NULL,
		);`, syncSanpshotTable)

	if _, err := s.db.Exec(createSnapshotTableQuery); err != nil {
		return err
	}

	createDataChunkTableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			begin INTEGER NOT NULL,
			end INTEGER NOT NULL,
			checksum INTEGER NOT NULL
		);`, syncDataChunkTable)

	_, err := s.db.Exec(createDataChunkTableQuery)
	return err
}

func (s *SyncDB) Close() {
	s.db.Close()
}

// SaveIntervalChecksum updates the checksum for a specific file interval
func (s *SyncDB) SaveIntervalChecksum(begin, end int64, checksum uint64) error {
	insert := fmt.Sprintf(`INSERT INTO %s (begin, end, checksum) VALUES (%d, %d, %d)`, syncDataChunkTable, begin, end, checksum)

	result, err := s.db.Exec(insert)
	if err != nil {
		return err
	}

	// Optional: Check if the row was actually updated
	rows, _ := result.RowsAffected()
	if rows == 0 {
		// If row doesn't exist, we might want to insert it or log a warning
		// This depends on whether SaveInterval was called before this
		return fmt.Errorf("interval not found for checksum update")
	}

	return nil
}

// GetIntervalChecksum retrieves the checksum for a specific file interval
func (s *SyncDB) GetIntervalChecksum(snapshotName string, begin, end int64) (uint64, error) {
	var checksum uint64
	query := fmt.Sprintf(`SELECT checksum FROM %s WHERE begin = %d AND end = %d`, syncDataChunkTable, begin, end)

	err := s.db.QueryRow(query, syncDataChunkTable, snapshotName, begin, end).Scan(&checksum)
	if err != nil {
		return 0, err
	}

	return checksum, nil
}
