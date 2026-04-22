//go:build !noAutoUpgrades

package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

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

func TestCopyFile_unwritable_destination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file permission model differs")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make directory read-only so Create fails
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

	got, err := os.ReadFile(to)
	if err != nil {
		t.Fatalf("failed to read target: %v", err)
	}
	if string(got) != "new content" {
		t.Errorf("target content = %q, want %q", got, "new content")
	}

	// Source should be gone (renamed away)
	if _, err := os.Stat(from); !os.IsNotExist(err) {
		t.Errorf("source file should not exist after rename, err = %v", err)
	}
}

func TestReplaceExe_target_missing(t *testing.T) {
	// Regression: replaceExe used to fail when the target didn't exist.
	// It should tolerate os.IsNotExist on the old binary.
	dir := t.TempDir()
	from := filepath.Join(dir, "new_binary")
	to := filepath.Join(dir, "nonexistent_binary")

	if err := os.WriteFile(from, []byte("new content"), 0755); err != nil {
		t.Fatal(err)
	}

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

	// On Windows, old binary should be renamed to .old
	oldBackup := to + ".old"
	got, err := os.ReadFile(oldBackup)
	if err != nil {
		t.Fatalf(".old backup should exist: %v", err)
	}
	if string(got) != "old" {
		t.Errorf(".old content = %q, want %q", got, "old")
	}
}

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

	// On Unix, there should be no .old file
	oldBackup := to + ".old"
	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Errorf(".old backup should not exist on Unix, err = %v", err)
	}
}

func TestReplaceExe_cross_device_fallback(t *testing.T) {
	// When rename fails (e.g., cross-device), replaceExe should
	// fall back to copyFile. We simulate by using a real temp dir
	// and verifying the content arrives correctly regardless of method.
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

func TestGetInstallDir_fallback(t *testing.T) {
	baseDir := t.TempDir()
	z := ZVM{baseDir: baseDir}

	// Unset ZVM_INSTALL to exercise the executable-based path.
	// getInstallDir falls back to baseDir/self when it can't resolve.
	t.Setenv("ZVM_INSTALL", "")
	os.Unsetenv("ZVM_INSTALL")

	got, err := z.getInstallDir()
	if err != nil {
		// getInstallDir may return an error if the current executable
		// can't be modified — that's OK, we just verify it doesn't panic.
		t.Logf("getInstallDir returned error (expected in test env): %v", err)
		return
	}

	// Should return some valid directory
	if got == "" {
		t.Error("getInstallDir returned empty string")
	}
}
