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
	Load() error
	Save() error
	Entries() []string
	Remove(adif string)
	Add(adif string) 
}
type backlogImpl struct {
	file string
	entries []string
	dirty bool 
}

func create(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}
	return os.Create(p)
}

func ensureFile(dbFile string) (string, error) {
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

func NewBacklog(spec string) (*backlogImpl, error) {
	filename, err := ensureFile(spec)
	if err != nil {
		return nil, err
	}
	entries := make([]string,0)
	return &backlogImpl{file: filename, entries: entries}, nil
}

func (b *backlogImpl) Add(adif string) {
	b.entries = append(b.entries, adif)
	b.dirty = true 
}

func (b *backlogImpl) Entries() []string {
	entries := make([]string,0)
	for _,e := range(b.entries) {
		entries = append(entries,e)
	}
	return entries
}


func (b *backlogImpl) Remove(adif string) {
	entries := make([]string, 0)
	for _, e := range b.entries {
		if e == adif {
		        b.dirty = true
			continue
		}
		entries = append(entries, e)
	}
	b.entries = entries
}

func (b *backlogImpl) Save() error {
	if !b.dirty {
		return nil
	}
	f, err := os.OpenFile(b.file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer f.Close()

	for _, line := range b.entries {
		_, err = f.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	b.dirty = false 
	return nil

}

func (b *backlogImpl) Load() error {
	f, err := os.Open(b.file)
	if err != nil {
		fmt.Printf("could not open file: %s", err.Error())
		return err
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
	b.entries = lines
	b.dirty = false
	return err
}

