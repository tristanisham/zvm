package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSymlinkExists(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "symlink.test")
	if err := os.WriteFile(target, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Error(err)
	}

	stat, err := os.Lstat(link)
	if err != nil {
		t.Errorf("%q: %s", err, stat.Name())
	}
}

func TestSetXDGExecutables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG executable links are Linux-only")
	}

	root := t.TempDir()
	versionsDir := filepath.Join(root, "state")
	binDir := filepath.Join(root, "bin")
	versionDir := filepath.Join(versionsDir, "0.15.1")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"zig", "zls"} {
		if err := os.WriteFile(filepath.Join(versionDir, name), []byte(name), 0755); err != nil {
			t.Fatal(err)
		}
	}

	zvm := ZVM{Directories: Directories{
		state:      versionsDir,
		bin:        binDir,
		useXDGSpec: true,
	}}
	if err := zvm.setBin("0.15.1"); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"zig", "zls"} {
		target, err := os.Readlink(filepath.Join(binDir, name))
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(versionDir, name)
		if target != want {
			t.Fatalf("%s target = %q, want %q", name, target, want)
		}
	}

	withoutZLS := filepath.Join(versionsDir, "master")
	if err := os.MkdirAll(withoutZLS, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(withoutZLS, "zig"), []byte("zig"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := zvm.setBin("master"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(binDir, "zls")); !os.IsNotExist(err) {
		t.Fatalf("managed ZLS link was not removed: %v", err)
	}
}

func TestSetXDGExecutablesLeavesUnmanagedExecutableAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG executable links are Linux-only")
	}

	root := t.TempDir()
	versionsDir := filepath.Join(root, "state")
	binDir := filepath.Join(root, "bin")
	versionDir := filepath.Join(versionsDir, "master")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "zig"), []byte("zvm zig"), 0755); err != nil {
		t.Fatal(err)
	}
	unmanaged := filepath.Join(binDir, "zig")
	if err := os.WriteFile(unmanaged, []byte("system zig"), 0755); err != nil {
		t.Fatal(err)
	}

	zvm := ZVM{Directories: Directories{
		state:      versionsDir,
		bin:        binDir,
		useXDGSpec: true,
	}}
	if err := zvm.setBin("master"); err == nil {
		t.Fatal("setBin replaced an executable not managed by ZVM")
	}

	contents, err := os.ReadFile(unmanaged)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "system zig" {
		t.Fatalf("unmanaged executable was modified: %q", contents)
	}
}
