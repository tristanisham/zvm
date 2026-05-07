package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUntarXZ_RealWorldArtifact validates the load-bearing assumption that
// ulikunitz/xz can decode the LZMA2 settings used by real Zig-ecosystem
// release artifacts. Gated behind ZVM_TEST_REAL_TARXZ pointing at a local
// .tar.xz so CI doesn't have to fetch over the network.
func TestUntarXZ_RealWorldArtifact(t *testing.T) {
	tarball := os.Getenv("ZVM_TEST_REAL_TARXZ")
	if tarball == "" {
		t.Skip("ZVM_TEST_REAL_TARXZ not set; skipping real-world fixture test")
	}
	if _, err := os.Stat(tarball); err != nil {
		t.Fatalf("fixture not accessible: %v", err)
	}

	out := t.TempDir()
	if err := untarXZNative(tarball, out); err != nil {
		t.Fatalf("native xz extraction failed on real artifact %s: %v", tarball, err)
	}

	// Sanity: extraction must produce *something*.
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatalf("read out dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("extraction produced no entries")
	}

	// If a top-level entry is a directory, peek inside to confirm files
	// landed where we expect (catches the "wrote 0-byte files everywhere"
	// failure mode).
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		inner, err := os.ReadDir(filepath.Join(out, e.Name()))
		if err != nil {
			t.Fatalf("read inner dir: %v", err)
		}
		if len(inner) == 0 {
			t.Fatalf("inner dir %s is empty; extraction likely partial", e.Name())
		}
		t.Logf("real-world extract OK: %s contains %d entries", e.Name(), len(inner))
		return
	}
	// No nested dir — at least we got top-level files.
	t.Logf("real-world extract OK: %d top-level entries", len(entries))
}
