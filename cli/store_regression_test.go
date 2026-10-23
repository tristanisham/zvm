// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDedupeRejectsReplacedSource(t *testing.T) {
	z := newStoreTestZVM(t, []storeTestFile{
		{path: "0.15.1/lib/std.zig", content: "original"},
		{path: "0.15.2/lib/std.zig", content: "original"},
	})
	first := z.testPath("0.15.1/lib/std.zig")
	orig := linkFile
	t.Cleanup(func() { linkFile = orig })
	changed := false
	linkFile = func(old, new string) error {
		if old == first && !changed {
			changed = true
			// Model an installer replacing the source after hashing, before linking.
			if err := os.Remove(old); err != nil {
				return err
			}
			if err := os.WriteFile(old, []byte("reinstalled content"), 0644); err != nil {
				return err
			}
		}
		return os.Link(old, new)
	}
	if _, err := z.Dedupe(nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(z.testPath("0.15.2/lib/std.zig"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Fatalf("untouched version corrupted: got %q", got)
	}
}

func TestStoreMutationProcess(t *testing.T) {
	base := os.Getenv("ZVM_TEST_STORE_BASE")
	if base == "" {
		return
	}
	z := &ZVM{baseDir: base}
	fmt.Println("ready")
	var err error
	switch os.Getenv("ZVM_TEST_STORE_ACTION") {
	case "dedupe":
		_, err = z.Dedupe(nil)
	case "uninstall":
		err = z.Uninstall("0.15.1")
	case "prune":
		_, err = z.PruneStore()
	default:
		t.Fatal("unknown store operation")
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestStoreMutationsWaitForLock(t *testing.T) {
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"dedupe", "uninstall", "prune"} {
		t.Run(action, func(t *testing.T) {
			z := newStoreTestZVM(t, []storeTestFile{{path: "0.15.1/zig", content: "zig", mode: 0755}})
			if _, err := z.Dedupe(nil); err != nil {
				t.Fatal(err)
			}
			unlock, err := z.lockStore()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(unlock)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, "-test.run=^TestStoreMutationProcess$")
			cmd.Env = append(os.Environ(), "ZVM_TEST_STORE_BASE="+z.baseDir, "ZVM_TEST_STORE_ACTION="+action)
			out, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			if ready, err := bufio.NewReader(out).ReadString('\n'); err != nil || ready != "ready\n" {
				t.Fatalf("child readiness = %q, %v", ready, err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				t.Fatalf("%s finished while the store was locked: %v", action, err)
			case <-time.After(100 * time.Millisecond):
			}
			unlock()
			if err := <-done; err != nil {
				t.Fatalf("%s failed after unlocking: %v", action, err)
			}
		})
	}
}

func TestInstallAutoDedupeReleasesLock(t *testing.T) {
	t.Setenv("ZVM_TARGET_OS", "windows")
	t.Setenv("ZVM_TARGET_ARCH", "x86_64")
	const stem = "zig-test-0.15.2"
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	f, err := w.Create(stem + "/zig.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("zig")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive.Bytes())
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			fmt.Fprintf(w, `{"0.15.2":{"x86_64-windows":{"tarball":%q,"shasum":"%x"}}}`, server.URL+"/"+stem+".zip", sum)
		case "/" + stem + ".zip":
			w.Write(archive.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	z := &ZVM{baseDir: t.TempDir(), Settings: DefaultSettings}
	z.Settings.VersionMapUrl = server.URL + "/index.json"
	z.Settings.AutoDedupe = true
	if _, err := z.Install("0.15.2", false, false, false, true); err != nil {
		t.Fatal(err)
	}
	if entries := storeEntries(t, z); len(entries) != 1 {
		t.Fatalf("auto-dedupe stored %d files, want 1", len(entries))
	}
}

func TestUseOnCdRejectsStore(t *testing.T) {
	z, _ := setupMultishell(t, "0.15.1")
	if err := z.setBin("0.15.1"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(z.storeDir(), 0755); err != nil {
		t.Fatal(err)
	}
	link, err := z.NewMultishellLink(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(MultishellEnv, link)
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "build.zig"), []byte("//! zvm-lock: store\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	var stdout, stderr bytes.Buffer
	z.UseOnCd(&stdout, &stderr)
	if got := readlink(t, link); got != filepath.Join(z.baseDir, "0.15.1") {
		t.Fatalf("shell switched to %q; stdout=%q stderr=%q", got, &stdout, &stderr)
	}
}
