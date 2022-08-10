package main

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type Backlog interface {
	Store(adif string) error
	Fetch() ([]string, error)
	Remove(adif string) error
	Close()
}
type backlogFile struct {
	file string
}

func create(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}
	return os.Create(p)
}

func openFile(dbFile string) (string, error) {
	usr, _ := user.Current()
	homeDir := usr.HomeDir

	if strings.HasPrefix(dbFile, "~/") {
		dbFile = filepath.Join(homeDir, (dbFile)[2:])
	}
	if _, err := os.Stat(dbFile); err != nil {
		file, err := create(dbFile)
		if err != nil {
			return "", err
		}
		file.Close()
	}

	return dbFile, nil

}

func newBacklogFile(spec string) (*backlogFile, error) {
	filename, err := openFile(spec)
	if err != nil {
		return nil, err
	}

	return &backlogFile{file: filename}, nil
}

func (b backlogFile) Store(adif string) error {
	f, err := os.OpenFile(b.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(adif + "\n")
	return err
}

func (b backlogFile) Remove(adif string) error {
	lines, err := b.Fetch()

	f, err := os.OpenFile(b.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer f.Close()

	for _, line := range lines {
		if line == adif {
			continue
		}

		_, err = f.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func (b backlogFile) Fetch() ([]string, error) {
	f, err := os.Open(b.file)
	if err != nil {
		fmt.Printf("could not open file: %s", err.Error())
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lines := make([]string, 0)

	adif := ""
	for scanner.Scan() {
		t := scanner.Text()
		if len(t) < 1 {
			continue
		}
		if strings.HasPrefix(t, "<adif_ver") {
			if len(adif) > 0 {
				lines = append(lines, adif)
			}
			adif = t
			continue
		} else if len(adif) > 0 {
			adif = adif + t
		}
	}
	if len(adif) > 0 {
		lines = append(lines, adif)
	}
	f.Close()
	return lines, err
}

func (b backlogFile) Close() {

}
