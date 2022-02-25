package main

import (
	"database/sql"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Backlog interface {
	Store(adif string) error
	Fetch() ([]string, error)
	Remove(adif string) error
	Close()
}
type backlogDb struct {
	db *sql.DB
}

func ensureTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS entries ( "adif" TEXT);`)
	return err
}

func create(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}
	return os.Create(p)
}

func openDb(dbFile string) (*sql.DB, error) {
	usr, _ := user.Current()
	homeDir := usr.HomeDir

	if strings.HasPrefix(dbFile, "~/") {
		dbFile = filepath.Join(homeDir, (dbFile)[2:])
	}
	if _, err := os.Stat(dbFile); err != nil {
		file, err := create(dbFile)
		if err != nil {
			return nil, err
		}
		file.Close()
	}

	db, err := sql.Open("sqlite3", dbFile)
	return db, err

}

func newBacklogDb(spec string) (*backlogDb, error) {
	db, err := openDb(spec)
	if err != nil {
		return nil, err
	}
	err = ensureTable(db)
	if err != nil {
		return nil, err
	}

	return &backlogDb{db: db}, nil
}

func (b backlogDb) Store(adif string) error {
	_, err := b.db.Exec("INSERT INTO entries(adif) values(?)", adif)
	return err
}
func (b backlogDb) Remove(adif string) error {
	_, err := b.db.Exec("DELETE FROM entries WHERE adif=?", adif)
	return err
}
func (b backlogDb) Fetch() ([]string, error) {
	var adifs []string
	rows, err := b.db.Query("SELECT adif FROM entries")
	if err != nil {
		return adifs, err
	}
	defer rows.Close()

	for rows.Next() {
		var adif string
		err = rows.Scan(&adif)
		if err != nil {
			return adifs, err
		}
		adifs = append(adifs, adif)
	}
	return adifs, nil
}
func (b backlogDb) Close() {
	b.db.Close()
}
