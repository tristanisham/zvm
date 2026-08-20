// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

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
	"github.com/tristanisham/zvm/cli/meta"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Alias creates or updates key when val is provided, or prints the existing
// alias when val is nil. Alias values must identify an installed Zig version.
func (z *ZVM) Alias(ctx context.Context, key string, val *string) error {
	if key == "" {
		return ErrMissingArgument
	}
	if err := z.aliasDatabaseError(); err != nil {
		return err
	}

	if val == nil {
		var alias Alias
		if err := z.db.WithContext(ctx).Where("key = ?", key).First(&alias).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidAlias
			}
			return fmt.Errorf("%w: %w", ErrBadDatabase, err)
		}

		z.PrintAliases([]Alias{alias})
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

	if err := z.db.WithContext(ctx).Where("key = ?", key).
		Assign(Alias{Value: resolvedVal}).
		FirstOrCreate(&Alias{Key: key}).Error; err != nil {
		return fmt.Errorf("%w: %w", ErrFailedAliasSave, err)
	}

	return nil
}

// ResolveAlias looks up key. A missing key is reported by returning ok=false.
func (z *ZVM) ResolveAlias(ctx context.Context, key string) (value string, ok bool, err error) {
	if key == "" {
		return "", false, nil
	}
	if err := z.aliasDatabaseError(); err != nil {
		return "", false, err
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

// DeleteAlias deletes key. Deleting a key that does not exist is successful.
func (z *ZVM) DeleteAlias(ctx context.Context, key string) error {
	if key == "" {
		return ErrMissingArgument
	}
	if err := z.aliasDatabaseError(); err != nil {
		return err
	}

	if err := z.db.WithContext(ctx).Unscoped().Where("key = ?", key).Delete(&Alias{}).Error; err != nil {
		return fmt.Errorf("%w: %w", ErrFailedAliasClear, err)
	}
	return nil
}

// ClearAliases deletes all aliases.
func (z *ZVM) ClearAliases(ctx context.Context) error {
	if err := z.aliasDatabaseError(); err != nil {
		return err
	}
	if err := z.db.WithContext(ctx).Unscoped().Where("1 = 1").Delete(&Alias{}).Error; err != nil {
		return fmt.Errorf("%w: %w", ErrFailedAliasClear, err)
	}
	return nil
}

// ListAliases returns all aliases ordered by key.
func (z *ZVM) ListAliases(ctx context.Context) ([]Alias, error) {
	if err := z.aliasDatabaseError(); err != nil {
		return nil, err
	}

	var aliases []Alias
	if err := z.db.WithContext(ctx).Order("key").Find(&aliases).Error; err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadDatabase, err)
	}
	return aliases, nil
}

func (z *ZVM) initializeDatabase() error {
	dbPath := filepath.Join(z.baseDir, "memory.db")
	if _, err := os.Stat(dbPath); errors.Is(err, fs.ErrNotExist) {
		log.Debug("no alias database found")
	}

	logLevel := logger.Silent
	if meta.Debug {
		logLevel = logger.Info
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return errors.Join(ErrBadDatabase, err)
	}
	if err := db.AutoMigrate(&Alias{}); err != nil {
		return errors.Join(ErrBadDatabase, err)
	}

	z.db = db
	return nil
}

func (z *ZVM) aliasDatabaseError() error {
	if z.dbErr != nil {
		return z.dbErr
	}
	if z.db == nil {
		return ErrBadDatabase
	}
	return nil
}

// Alias is a user-defined name for an installed Zig version.
type Alias struct {
	gorm.Model
	Key   string `gorm:"uniqueIndex" json:"key"`
	Value string `json:"value"`
}

// PrintAliases prints aliases one per line.
func (z *ZVM) PrintAliases(aliases []Alias) {
	for _, alias := range aliases {
		if z.Settings.UseColor {
			fmt.Printf("%s %s\n", clr.Blue(alias.Key), clr.White(alias.Value))
		} else {
			fmt.Printf("%s %s\n", alias.Key, alias.Value)
		}
	}
}
