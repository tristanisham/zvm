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

	val := "v0.13.0"
	if err := z.Alias(ctx, "stable", &val); err != nil {
		t.Fatalf("save alias: %v", err)
	}

	got, ok, err := z.ResolveAlias(ctx, "stable")
	if err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	if !ok {
		t.Fatal("expected alias to exist")
	}
	if got != "0.13.0" {
		t.Fatalf("expected normalized alias value %q, got %q", "0.13.0", got)
	}
}

func TestAliasSaveVersionShorthandResolvesAndStoresFullVersion(t *testing.T) {
	ctx := context.Background()
	z := newTestAliasZVM(t)
	installTestVersion(t, z, "0.16.0")
	installTestVersion(t, z, "0.16.1")

	val := ".16"
	if err := z.Alias(ctx, "sixteen", &val); err != nil {
		t.Fatalf("save alias: %v", err)
	}

	got, ok, err := z.ResolveAlias(ctx, "sixteen")
	if err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	if !ok {
		t.Fatal("expected alias to exist")
	}
	if got != "0.16.1" {
		t.Fatalf("expected resolved alias value %q, got %q", "0.16.1", got)
	}
}

func TestAliasSaveNonInstalledVersionReturnsInvalidAliasValueAndDoesNotSave(t *testing.T) {
	ctx := context.Background()
	z := newTestAliasZVM(t)

	val := "9.9.9"
	err := z.Alias(ctx, "missing", &val)
	if !errors.Is(err, ErrInvalidAliasValue) {
		t.Fatalf("expected ErrInvalidAliasValue, got %v", err)
	}

	got, ok, err := z.ResolveAlias(ctx, "missing")
	if err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	if ok {
		t.Fatalf("expected alias not to be saved, got %q", got)
	}
}

func TestResolveAliasMissingIsNotAnError(t *testing.T) {
	ctx := context.Background()
	z := newTestAliasZVM(t)

	got, ok, err := z.ResolveAlias(ctx, "missing")
	if err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	if ok {
		t.Fatalf("expected missing alias, got %q", got)
	}
	if got != "" {
		t.Fatalf("expected empty value for missing alias, got %q", got)
	}
}
