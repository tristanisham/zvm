// Tests for the cli/meta package: version format, sentinel errors,
// build flags, and the platform-specific Link function.
//
// Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>

package meta

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestVERSION_is_valid_semver checks that the VERSION constant follows
// the "vMAJOR.MINOR.PATCH" format expected by Go's semver package.
func TestVERSION_is_valid_semver(t *testing.T) {
	if !strings.HasPrefix(VERSION, "v") {
		t.Errorf("VERSION should start with 'v', got %q", VERSION)
	}
	parts := strings.Split(strings.TrimPrefix(VERSION, "v"), ".")
	if len(parts) != 3 {
		t.Errorf("VERSION should have 3 dot-separated parts, got %d in %q", len(parts), VERSION)
	}
}

// TestVerCopy_contains_version_and_platform verifies that VerCopy includes
// both the version string and "GOOS/GOARCH" so users see full build info.
func TestVerCopy_contains_version_and_platform(t *testing.T) {
	if !strings.Contains(VerCopy, VERSION) {
		t.Errorf("VerCopy should contain VERSION %q, got %q", VERSION, VerCopy)
	}
	expected := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	if !strings.Contains(VerCopy, expected) {
		t.Errorf("VerCopy should contain %q, got %q", expected, VerCopy)
	}
}

// TestSentinelErrors_are_distinct ensures that none of the sentinel errors
// in errors.go accidentally alias each other via errors.Is.
func TestSentinelErrors_are_distinct(t *testing.T) {
	errs := []struct {
		name string
		err  error
	}{
		{"ErrWinEscToAdmin", ErrWinEscToAdmin},
		{"ErrEscalatedSymlink", ErrEscalatedSymlink},
		{"ErrEscalatedHardlink", ErrEscalatedHardlink},
	}

	for i, a := range errs {
		for j, b := range errs {
			if i != j && errors.Is(a.err, b.err) {
				t.Errorf("%s and %s should not match", a.name, b.name)
			}
		}
	}
}

// TestSentinelErrors_wrap_correctly verifies that sentinel errors survive
// wrapping with fmt.Errorf %w and can still be matched with errors.Is.
func TestSentinelErrors_wrap_correctly(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", ErrWinEscToAdmin)
	if !errors.Is(wrapped, ErrWinEscToAdmin) {
		t.Error("wrapped error should match ErrWinEscToAdmin")
	}
}

// TestNoAutoUpgrades_default confirms that in a normal build (without the
// noAutoUpgrades build tag), the flag is false so self-upgrade is enabled.
func TestNoAutoUpgrades_default(t *testing.T) {
	if NoAutoUpgrades {
		t.Error("NoAutoUpgrades should be false in default build")
	}
}

// TestLink_creates_link verifies that Link creates a working junction
// (Windows) or symlink (Unix) that allows reading files through it.
func TestLink_creates_link(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a marker file inside the target directory
	marker := filepath.Join(target, "marker.txt")
	if err := os.WriteFile(marker, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(dir, "link")
	if err := Link(target, linkPath); err != nil {
		if runtime.GOOS == "windows" {
			t.Fatalf("Link (junction) failed: %v", err)
		}
		t.Fatalf("Link (symlink) failed: %v", err)
	}
	defer os.Remove(linkPath)

	// Verify the link works by reading the marker through it
	got, err := os.ReadFile(filepath.Join(linkPath, "marker.txt"))
	if err != nil {
		t.Fatalf("could not read through link: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("marker content = %q, want %q", got, "hello")
	}
}

// TestLink_fails_on_existing_target verifies that Link returns an error
// if the link path already exists, rather than silently overwriting.
func TestLink_fails_on_existing_target(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(dir, "link")
	// Create the link once
	if err := Link(target, linkPath); err != nil {
		t.Fatalf("first Link failed: %v", err)
	}
	defer os.Remove(linkPath)

	// Creating the same link again should fail
	if err := Link(target, linkPath); err == nil {
		t.Error("expected error when link target already exists")
	}
}
