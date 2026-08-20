package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTestAliasZVM(t *testing.T) *ZVM {
	t.Helper()

	z := &ZVM{baseDir: t.TempDir()}
	if err := z.initializeDatabase(); err != nil {
		t.Fatalf("initialize database: %v", err)
	}
	return z
}

func installTestVersion(t *testing.T, z *ZVM, version string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(z.baseDir, version), 0755); err != nil {
		t.Fatalf("create installed version: %v", err)
	}
}

func TestAliasSaveInstalledVersionNormalizesValue(t *testing.T) {
	ctx := context.Background()
	z := newTestAliasZVM(t)
	installTestVersion(t, z, "0.13.0")

	value := "v0.13.0"
	if err := z.Alias(ctx, "work", &value); err != nil {
		t.Fatalf("save alias: %v", err)
	}

	got, ok, err := z.ResolveAlias(ctx, "work")
	if err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	if !ok || got != "0.13.0" {
		t.Fatalf("ResolveAlias() = %q, %v; want %q, true", got, ok, "0.13.0")
	}
}

func TestAliasSaveVersionShorthandResolvesLatestInstalledVersion(t *testing.T) {
	ctx := context.Background()
	z := newTestAliasZVM(t)
	installTestVersion(t, z, "0.16.0")
	installTestVersion(t, z, "0.16.1")

	value := ".16"
	if err := z.Alias(ctx, "sixteen", &value); err != nil {
		t.Fatalf("save alias: %v", err)
	}

	got, ok, err := z.ResolveAlias(ctx, "sixteen")
	if err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	if !ok || got != "0.16.1" {
		t.Fatalf("ResolveAlias() = %q, %v; want %q, true", got, ok, "0.16.1")
	}
}

func TestAliasSaveNonInstalledVersionDoesNotSave(t *testing.T) {
	ctx := context.Background()
	z := newTestAliasZVM(t)

	value := "9.9.9"
	if err := z.Alias(ctx, "missing", &value); !errors.Is(err, ErrInvalidAliasValue) {
		t.Fatalf("Alias() error = %v; want ErrInvalidAliasValue", err)
	}

	got, ok, err := z.ResolveAlias(ctx, "missing")
	if err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	if ok {
		t.Fatalf("expected alias not to be saved, got %q", got)
	}
}

func TestDeleteAndClearAliases(t *testing.T) {
	ctx := context.Background()
	z := newTestAliasZVM(t)
	installTestVersion(t, z, "0.13.0")
	value := "0.13.0"

	for _, name := range []string{"work", "play"} {
		if err := z.Alias(ctx, name, &value); err != nil {
			t.Fatalf("save alias %q: %v", name, err)
		}
	}
	if err := z.DeleteAlias(ctx, "work"); err != nil {
		t.Fatalf("delete alias: %v", err)
	}
	if _, ok, err := z.ResolveAlias(ctx, "work"); err != nil || ok {
		t.Fatalf("deleted alias ResolveAlias() ok=%v, err=%v; want false, nil", ok, err)
	}
	if err := z.Alias(ctx, "work", &value); err != nil {
		t.Fatalf("recreate deleted alias: %v", err)
	}
	if err := z.ClearAliases(ctx); err != nil {
		t.Fatalf("clear aliases: %v", err)
	}
	aliases, err := z.ListAliases(ctx)
	if err != nil {
		t.Fatalf("list aliases: %v", err)
	}
	if len(aliases) != 0 {
		t.Fatalf("ListAliases() returned %d aliases after clear; want 0", len(aliases))
	}
}
