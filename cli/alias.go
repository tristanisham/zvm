package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/libtnb/sqlite"
	"github.com/tristanisham/clr"
	"gorm.io/gorm"
)

func (z *ZVM) Alias(ctx context.Context, key string, val *string) error {
	if key == "" {
		return ErrMissingArgument
	}

	if val == nil {
		var alias Alias
		if err := z.db.WithContext(ctx).Where("key = ?", key).First(&alias).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidAlias
			}
			return fmt.Errorf("%w: %w", ErrBadDatabase, err)
		}

		PrintAliases([]Alias{alias})
		return nil
	}

	normalizedVal := strings.TrimPrefix(*val, "v")
	installedVersions, err := z.GetInstalledVersions()
	if err != nil {
		return err
	}

	resolvedVal, err := resolveVersionShorthand(normalizedVal, installedVersions)
	if err != nil {
		return err
	}

	if !slices.Contains(installedVersions, resolvedVal) {
		return ErrInvalidAliasValue
	}

	if err := z.db.WithContext(ctx).Where("key = ?", key).Assign(Alias{Value: resolvedVal}).FirstOrCreate(&Alias{Key: key}).Error; err != nil {
		return fmt.Errorf("%w: %w", ErrFailedAliasSave, err)
	}

	return nil
}

func (z *ZVM) ResolveAlias(ctx context.Context, key string) (string, bool, error) {
	if key == "" {
		return "", false, nil
	}

	var alias Alias
	if err := z.db.WithContext(ctx).Where("key = ?", key).First(&alias).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("%w: %w", ErrBadDatabase, err)
	}

	return alias.Value, true, nil
}

func (z *ZVM) DeleteAlias(ctx context.Context, key string) error {
	if key == "" {
		return ErrMissingArgument
	}

	if err := z.db.WithContext(ctx).Where("key = ?", key).Delete(&Alias{}).Error; err != nil {
		return fmt.Errorf("%w: %w", ErrFailedAliasClear, err)
	}

	return nil
}

func (z *ZVM) ClearAliases(ctx context.Context) error {
	if err := z.db.WithContext(ctx).Where("1 = 1").Delete(&Alias{}).Error; err != nil {
		return fmt.Errorf("%w: %w", ErrFailedAliasClear, err)
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
	Key   string `gorm:"index" json:"key"`
	Value string `gorm:"index" json:"value"`
}

func PrintAliases(aliases []Alias) {
	for _, v := range aliases {
		fmt.Printf("%s %s\n", clr.Blue(v.Key), clr.White(v.Value))
	}
}
func NewAlias(key, val string) *Alias {
	return &Alias{Key: key, Value: val}
}
