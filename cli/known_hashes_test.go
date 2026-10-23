package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

func TestKnownHashesRejectsNullStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), knownHashesFile)
	if err := os.WriteFile(path, []byte("null\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := recordKnownHash(path, "zig-test.tar.xz", "aa"); !errors.Is(err, ErrBadKnownHashes) {
		t.Fatalf("recordKnownHash() = %v; want ErrBadKnownHashes", err)
	}
}

func TestRecordKnownHashRejectsConcurrentReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), knownHashesFile)
	const name = "zig-test.tar.xz"
	start := make(chan struct{})
	results := make(chan error, 2)
	var writers sync.WaitGroup
	for _, sum := range []string{"aa", "bb"} {
		writers.Go(func() {
			<-start
			results <- recordKnownHash(path, name, sum)
		})
	}
	close(start)
	writers.Wait()
	close(results)
	succeeded, rejected := 0, 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrKnownHashMismatch):
			rejected++
		default:
			t.Fatal(err)
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("succeeded=%d, rejected=%d; want one of each", succeeded, rejected)
	}
}

// Each child runs the real store update in a separate process, so a Go mutex
// alone cannot make the concurrent-writer regression test pass.
func TestKnownHashWriterProcess(t *testing.T) {
	path := os.Getenv("ZVM_TEST_HASH_STORE")
	if path == "" {
		return
	}
	name := os.Getenv("ZVM_TEST_HASH_NAME")
	if err := recordKnownHash(path, name, "aa"); err != nil {
		t.Fatal(err)
	}
}

func TestKnownHashesPreservesConcurrentProcessWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), knownHashesFile)
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	const count = 16
	start := make(chan struct{})
	results := make(chan error, count)
	var writers sync.WaitGroup
	for i := range count {
		writers.Go(func() {
			<-start
			cmd := exec.Command(bin, "-test.run=^TestKnownHashWriterProcess$")
			cmd.Env = append(os.Environ(), "ZVM_TEST_HASH_STORE="+path,
				fmt.Sprintf("ZVM_TEST_HASH_NAME=zig-platform-%d.tar.xz", i))
			output, err := cmd.CombinedOutput()
			if err != nil {
				results <- fmt.Errorf("writer %d: %w: %s", i, err, output)
				return
			}
			results <- nil
		})
	}
	close(start)
	writers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	hashes, err := loadKnownHashes(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range count {
		name := fmt.Sprintf("zig-platform-%d.tar.xz", i)
		if hashes[name] != "aa" {
			t.Errorf("lost pin for %s", name)
		}
	}
}
