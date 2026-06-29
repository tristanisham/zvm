package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/libtnb/sqlite"
	"gorm.io/gorm"
)

func (z *ZVM) Alias() error {
	return nil
}

func (z *ZVM) initializeDatabase() error {
	dbPath := filepath.Join(z.baseDir, "memory.db")
	if _, stat := os.Stat(dbPath); errors.Is(stat, fs.ErrNotExist) {
		log.Debug("No database found")
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return errors.Join(ErrBadDatabase, err)
	}

	db.AutoMigrate(&Alias{})

	z.db = db

	return nil
}

type Alias struct {
	gorm.Model
	Name  string `gorm:"index"`
	Value string `gorm:"index"`
}
