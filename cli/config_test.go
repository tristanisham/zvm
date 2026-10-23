package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInitializeWritesSettingsWithReadableMode(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("ZVM_PATH", baseDir)

	Initialize()

	info, err := os.Stat(filepath.Join(baseDir, "settings.json"))
	if err != nil {
		t.Fatalf("stat settings.json: %v", err)
	}
	if got := info.Mode().Perm(); runtime.GOOS != "windows" && got != 0o644 {
		t.Fatalf("settings.json mode = %o, want 644", got)
	}
}

func TestValidateVMUalias(t *testing.T) {
	if !validVmuAlis("mach") {
		t.Errorf("mach: should be true")
	}

	if !validVmuAlis("default") {
		t.Errorf("default: should be true")
	}
}
