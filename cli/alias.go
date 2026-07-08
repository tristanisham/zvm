package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/libtnb/sqlite"
	"github.com/tristanisham/clr"
	"gorm.io/gorm"
)

func (z *ZVM) Alias(ctx context.Context, key string, val *string) error {
	if key != "" {
		if val == nil {
			gorm.G[Alias](z.db).Select("where name = ?", key).Find(ctx)
		} else {
			if *val == "" {
				if err := z.db.Delete(&Alias{Name: key}); err.Error != nil {
					
				}
			} else {
				if err := z.db.FirstOrCreate(NewAlias(key, *val)); err.Error != nil {
					return fmt.Errorf("%w. %w", ErrFailedAliasSave, err.Error)
				}
			}
			
		}
	}

	return nil
}

func (z *ZVM) ListAliases(ctx context.Context) ([]Alias, error) {
	return gorm.G[Alias](z.db).Find(ctx)
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

func PrintAliases(aliases []Alias) {
	fmt.Printf("%s %s\n", clr.Blue("key"), clr.White("value"))
	for _, v := range aliases {
		fmt.Printf("\t%s %s\n", clr.Blue(v.Name), clr.White(v.Value))
	}
}
func NewAlias(key, val string) *Alias {
	return &Alias{Name: key, Value: val}
}
