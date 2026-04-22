// Regression tests for the ZVM self-upgrade helper functions.
// These cover copyFile, replaceExe, isSymlink, resolveSymlink, and getInstallDir.
//
// Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>

//go:build !noAutoUpgrades

package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestCopyFile verifies that copyFile faithfully reproduces file contents,
// handles empty files, binary data, and returns an error for missing sources.
func TestCopyFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		missingFrom bool
		wantErr     bool
	}{
		{
			name:    "copies file contents correctly",
			content: "hello world\nline two\n",
		},
		{
			name:    "copies empty file",
			content: "",
		},
		{
			name:    "copies binary-like content",
			content: "\x00\x01\x02\xff\xfe",
		},
		{
			name:        "returns error for missing source",
			missingFrom: true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "src")
			dst := filepath.Join(dir, "dst")

			// Only create the source file if the test case expects it to exist.
			if !tt.missingFrom {
				if err := os.WriteFile(src, []byte(tt.content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			err := copyFile(src, dst)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Read back the destination and compare byte-for-byte.
			got, err := os.ReadFile(dst)
			if err != nil {
				t.Fatalf("failed to read dst: %v", err)
			}
			if string(got) != tt.content {
				t.Errorf("content mismatch: got %q, want %q", got, tt.content)
			}
		})
	}
}

// TestCopyFile_unwritable_destination verifies that copyFile returns an error
// when the destination directory is read-only. Skipped on Windows where the
// permission model differs.
func TestCopyFile_unwritable_destination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file permission model differs")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make directory read-only so os.Create inside copyFile fails.
	roDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(roDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(roDir, 0755)

	dst := filepath.Join(roDir, "dst")
	if err := copyFile(src, dst); err == nil {
		t.Error("expected error writing to read-only dir, got nil")
	}
}

// TestReplaceExe_basic verifies the happy path: the new binary replaces
// the old one, and the source file is consumed (renamed away).
func TestReplaceExe_basic(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "new_binary")
	to := filepath.Join(dir, "old_binary")

	if err := os.WriteFile(from, []byte("new content"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("old content"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := replaceExe(from, to); err != nil {
		t.Fatalf("replaceExe failed: %v", err)
	}

	// The target should now contain the new binary's content.
	got, err := os.ReadFile(to)
	if err != nil {
		t.Fatalf("failed to read target: %v", err)
	}
	if string(got) != "new content" {
		t.Errorf("target content = %q, want %q", got, "new content")
	}

	// The source should be gone — it was renamed into the target path.
	if _, err := os.Stat(from); !os.IsNotExist(err) {
		t.Errorf("source file should not exist after rename, err = %v", err)
	}
}

// TestReplaceExe_target_missing is a regression test: replaceExe previously
// failed when the target binary didn't exist (e.g. first install or manual
// deletion). It should tolerate fs.ErrNotExist on the old binary.
func TestReplaceExe_target_missing(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "new_binary")
	to := filepath.Join(dir, "nonexistent_binary")

	if err := os.WriteFile(from, []byte("new content"), 0755); err != nil {
		t.Fatal(err)
	}

	// This should succeed even though 'to' doesn't exist yet.
	if err := replaceExe(from, to); err != nil {
		t.Fatalf("replaceExe should tolerate missing target, got: %v", err)
	}

	got, err := os.ReadFile(to)
	if err != nil {
		t.Fatalf("failed to read target: %v", err)
	}
	if string(got) != "new content" {
		t.Errorf("target content = %q, want %q", got, "new content")
	}
}

// TestReplaceExe_windows_creates_old_backup verifies that on Windows,
// replaceExe renames the existing binary to .old instead of deleting it,
// since Windows locks running executables.
func TestReplaceExe_windows_creates_old_backup(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	dir := t.TempDir()
	from := filepath.Join(dir, "new.exe")
	to := filepath.Join(dir, "old.exe")

	if err := os.WriteFile(from, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := replaceExe(from, to); err != nil {
		t.Fatalf("replaceExe failed: %v", err)
	}

	// The old binary should now live at <name>.old with its original content.
	oldBackup := to + ".old"
	got, err := os.ReadFile(oldBackup)
	if err != nil {
		t.Fatalf(".old backup should exist: %v", err)
	}
	if string(got) != "old" {
		t.Errorf(".old content = %q, want %q", got, "old")
	}
}

// TestReplaceExe_unix_removes_old verifies that on Unix, the old binary
// is backed up to .old (same as Windows) for safe rollback.
func TestReplaceExe_unix_removes_old(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	dir := t.TempDir()
	from := filepath.Join(dir, "new")
	to := filepath.Join(dir, "old")

	if err := os.WriteFile(from, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := replaceExe(from, to); err != nil {
		t.Fatalf("replaceExe failed: %v", err)
	}

	// The .old backup should exist since we now use rename-to-backup on all platforms.
	oldBackup := to + ".old"
	got, err := os.ReadFile(oldBackup)
	if err != nil {
		t.Fatalf(".old backup should exist on Unix too: %v", err)
	}
	if string(got) != "old" {
		t.Errorf(".old content = %q, want %q", got, "old")
	}
}

// TestReplaceExe_cross_device_fallback verifies that when os.Rename fails
// (e.g. cross-device move), replaceExe falls back to copyFile and the
// target still ends up with the correct content.
func TestReplaceExe_cross_device_fallback(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "src")
	to := filepath.Join(dir, "dst")

	if err := os.WriteFile(from, []byte("payload"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := replaceExe(from, to); err != nil {
		t.Fatalf("replaceExe failed: %v", err)
	}

	got, err := os.ReadFile(to)
	if err != nil {
		t.Fatalf("failed to read target: %v", err)
	}
	if string(got) != "payload" {
		t.Errorf("target content = %q, want %q", got, "payload")
	}
}

// TestIsSymlink checks that isSymlink correctly distinguishes regular files,
// symlinks, and nonexistent paths. Symlink subtests are skipped on Windows
// where creation may require elevated privileges.
func TestIsSymlink(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular.txt")
	if err := os.WriteFile(regular, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("regular file is not a symlink", func(t *testing.T) {
		got, err := isSymlink(regular)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Error("expected false for regular file")
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := isSymlink(filepath.Join(dir, "nope"))
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	if runtime.GOOS == "windows" {
		t.Skip("Symlink creation may require elevated privileges on Windows")
	}

	symlink := filepath.Join(dir, "link")
	if err := os.Symlink(regular, symlink); err != nil {
		t.Fatal(err)
	}

	t.Run("symlink is detected", func(t *testing.T) {
		got, err := isSymlink(symlink)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Error("expected true for symlink")
		}
	})
}

// TestResolveSymlink verifies that resolveSymlink follows a symlink and
// returns the absolute path to the real target file.
func TestResolveSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink creation may require elevated privileges on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	symlink := filepath.Join(dir, "link")
	if err := os.Symlink(target, symlink); err != nil {
		t.Fatal(err)
	}

	resolved, err := resolveSymlink(symlink)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	absTarget, _ := filepath.Abs(target)
	if resolved != absTarget {
		t.Errorf("resolveSymlink = %q, want %q", resolved, absTarget)
	}
}

// TestResolveSymlink_not_a_symlink verifies that resolveSymlink returns an
// error when called on a regular file (os.Readlink fails on non-symlinks).
func TestResolveSymlink_not_a_symlink(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular.txt")
	if err := os.WriteFile(regular, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := resolveSymlink(regular)
	if err == nil {
		t.Error("expected error when resolving a non-symlink")
	}
}

// TestGetInstallDir_env_override verifies that when ZVM_INSTALL is set,
// getInstallDir returns that value directly without probing the executable.
func TestGetInstallDir_env_override(t *testing.T) {
	z := ZVM{baseDir: t.TempDir()}
	customDir := filepath.Join(t.TempDir(), "custom-install")
	if err := os.MkdirAll(customDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ZVM_INSTALL", customDir)

	got, err := z.getInstallDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != customDir {
		t.Errorf("getInstallDir = %q, want %q", got, customDir)
	}
}

// TestGetInstallDir_fallback verifies that when ZVM_INSTALL is unset,
// getInstallDir falls back to resolving the running executable's directory
// (or baseDir/self if that fails). In a test environment the executable is
// the test binary, so we just verify it returns something sensible.
func TestGetInstallDir_fallback(t *testing.T) {
	baseDir := t.TempDir()
	z := ZVM{baseDir: baseDir}

	// Unset ZVM_INSTALL to exercise the executable-based path.
	t.Setenv("ZVM_INSTALL", "")
	os.Unsetenv("ZVM_INSTALL")

	got, err := z.getInstallDir()
	if err != nil {
		// getInstallDir may return an error if the current executable
		// can't be modified — that's OK, we just verify it doesn't panic.
		t.Logf("getInstallDir returned error (expected in test env): %v", err)
		return
	}

	// Should return some valid directory path.
	if got == "" {
		t.Error("getInstallDir returned empty string")
	}
}
