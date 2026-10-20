// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jedisct1/go-minisign"
	"github.com/ulikunitz/xz"
)

func TestSignedFilename(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    string
		wantErr error
	}{
		{"zig format", "trusted comment: timestamp:1760215991\tfile:zig-x86_64-linux-0.15.2.tar.xz\thashed", "zig-x86_64-linux-0.15.2.tar.xz", nil},
		{"dev build", "trusted comment: timestamp:1792322638\tfile:zig-x86_64-linux-0.18.0-dev.349+d924f83bf.tar.xz\thashed", "zig-x86_64-linux-0.18.0-dev.349+d924f83bf.tar.xz", nil},
		{"file only", "trusted comment: file:a.zip", "a.zip", nil},
		{"extra unknown fields", "trusted comment: build:42 timestamp:1\tfile:a.tar.xz\tsigner:zsf\thashed", "a.tar.xz", nil},
		{"weird whitespace", "trusted comment:   timestamp:1 \t\t file:a.tar.xz  \t", "a.tar.xz", nil},
		{"missing file field", "trusted comment: timestamp:1760215991\thashed", "", ErrMissingSignedFilename},
		{"empty file field", "trusted comment: timestamp:1\tfile:", "", ErrMissingSignedFilename},
		{"duplicate file field", "trusted comment: file:a.tar.xz\tfile:b.tar.xz", "", ErrMissingSignedFilename},
		{"field name is prefix-matched only", "trusted comment: myfile:a.tar.xz", "", ErrMissingSignedFilename},
		{"missing prefix", "timestamp:1\tfile:a.tar.xz", "", ErrMissingSignedFilename},
		{"empty", "", "", ErrMissingSignedFilename},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := signedFilename(tt.comment)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("signedFilename(%q) error = %v, want %v", tt.comment, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("signedFilename(%q) = %q, want %q", tt.comment, got, tt.want)
			}
		})
	}
}

func TestCheckSignedFilename(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    string
		wantErr error
	}{
		{"match", "trusted comment: timestamp:1\tfile:zig-x86_64-linux-0.15.2.tar.xz\thashed", "zig-x86_64-linux-0.15.2.tar.xz", nil},
		{"older release (downgrade)", "trusted comment: timestamp:1\tfile:zig-x86_64-linux-0.14.1.tar.xz\thashed", "zig-x86_64-linux-0.15.2.tar.xz", ErrSignedFilenameMismatch},
		{"other platform", "trusted comment: timestamp:1\tfile:zig-aarch64-linux-0.15.2.tar.xz\thashed", "zig-x86_64-linux-0.15.2.tar.xz", ErrSignedFilenameMismatch},
		{"path instead of basename", "trusted comment: file:../zig-x86_64-linux-0.15.2.tar.xz", "zig-x86_64-linux-0.15.2.tar.xz", ErrSignedFilenameMismatch},
		{"case differs", "trusted comment: file:ZIG-x86_64-linux-0.15.2.tar.xz", "zig-x86_64-linux-0.15.2.tar.xz", ErrSignedFilenameMismatch},
		{"missing", "trusted comment: timestamp:1", "zig-x86_64-linux-0.15.2.tar.xz", ErrMissingSignedFilename},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSignedFilename(minisign.Signature{TrustedComment: tt.comment}, tt.want)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("checkSignedFilename() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTarballMatchesVersion(t *testing.T) {
	tests := []struct {
		tarName, version string
		want             bool
	}{
		{"zig-x86_64-linux-0.15.2.tar.xz", "0.15.2", true},
		{"zig-linux-x86_64-0.11.0.tar.xz", "0.11.0", true},
		{"zig-x86_64-windows-0.15.2.zip", "0.15.2", true},
		{"zig-win64-0.2.0.zip", "0.2.0", true},
		{"zig-x86_64-linux-0.18.0-dev.349+d924f83bf.tar.xz", "0.18.0-dev.349+d924f83bf", true},
		{"zig-x86_64-linux-0.14.1.tar.xz", "0.15.2", false},
		{"zig-x86_64-linux-10.15.2.tar.xz", "0.15.2", false},
		{"zig-x86_64-linux-0.15.2.tar.xz", "", false},
		{"evil-x86_64-linux-0.15.2.tar.xz", "0.15.2", false},
	}

	for _, tt := range tests {
		if got := tarballMatchesVersion(tt.tarName, tt.version); got != tt.want {
			t.Errorf("tarballMatchesVersion(%q, %q) = %v, want %v", tt.tarName, tt.version, got, tt.want)
		}
	}
}

func TestKnownHashes(t *testing.T) {
	const name = "zig-x86_64-linux-0.15.2.tar.xz"
	tests := []struct {
		name     string
		existing string // file contents; empty means no file
		sum      string
		recorded bool
		wantErr  error
	}{
		{"first use", "", "aa", false, nil},
		{"same hash", `{"` + name + `": "aa"}`, "aa", true, nil},
		{"different hash", `{"` + name + `": "aa"}`, "bb", true, ErrKnownHashMismatch},
		{"other file pinned", `{"zig-x86_64-linux-0.14.1.tar.xz": "aa"}`, "bb", false, nil},
		{"corrupt store", `{not json`, "aa", false, ErrBadKnownHashes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), knownHashesFile)
			if tt.existing != "" {
				if err := os.WriteFile(path, []byte(tt.existing), 0644); err != nil {
					t.Fatal(err)
				}
			}
			recorded, err := checkKnownHash(path, name, tt.sum)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("checkKnownHash() error = %v, want %v", err, tt.wantErr)
			}
			if recorded != tt.recorded {
				t.Fatalf("checkKnownHash() recorded = %v, want %v", recorded, tt.recorded)
			}
		})
	}

	t.Run("record then reject different hash", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), knownHashesFile)
		if err := recordKnownHash(path, name, "aa"); err != nil {
			t.Fatal(err)
		}
		if err := recordKnownHash(path, "other.tar.xz", "cc"); err != nil {
			t.Fatal(err)
		}
		if _, err := checkKnownHash(path, name, "aa"); err != nil {
			t.Fatalf("same hash rejected: %v", err)
		}
		if _, err := checkKnownHash(path, name, "bb"); !errors.Is(err, ErrKnownHashMismatch) {
			t.Fatalf("different hash: error = %v, want %v", err, ErrKnownHashMismatch)
		}
	})
}

// The pin store lives in the base directory next to installed versions; it
// must not be mistaken for a version or swept up by Clean.
func TestKnownHashesFileIsNotAVersion(t *testing.T) {
	z := &ZVM{baseDir: t.TempDir()}
	installTestVersion(t, z, "0.15.2")
	if err := recordKnownHash(filepath.Join(z.baseDir, knownHashesFile), "a.tar.xz", "aa"); err != nil {
		t.Fatal(err)
	}

	versions, err := z.GetInstalledVersions()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(versions, []string{"0.15.2"}) {
		t.Fatalf("GetInstalledVersions() = %v, want [0.15.2]", versions)
	}
	if err := z.Clean(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(z.baseDir, knownHashesFile)); err != nil {
		t.Fatalf("Clean removed the known hashes store: %v", err)
	}
}

func TestMirrorCandidates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Comments, blank lines, CRLF and no trailing newline.
		fmt.Fprint(w, "# comment\nhttps://a.example/zig\r\n\n  https://b.example/  \nhttps://c.example")
	}))
	defer srv.Close()
	stubShuffle(t)

	tarURL := "https://ziglang.org/download/0.15.2/zig-x86_64-linux-0.15.2.tar.xz"
	got := mirrorCandidates(srv.URL, tarURL)
	want := []string{
		"https://a.example/zig/zig-x86_64-linux-0.15.2.tar.xz",
		"https://b.example/zig-x86_64-linux-0.15.2.tar.xz",
		"https://c.example/zig-x86_64-linux-0.15.2.tar.xz",
		tarURL,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("mirrorCandidates() = %v, want %v", got, want)
	}

	// An unreachable mirror list falls back to ziglang.org alone.
	if got := mirrorCandidates(srv.URL+"/missing\x00", tarURL); !slices.Equal(got, []string{tarURL}) {
		t.Fatalf("mirrorCandidates(bad list) = %v, want [%s]", got, tarURL)
	}
}

// --- signing fixtures ----------------------------------------------------

type testSigner struct {
	sk     minisign.PrivateKey
	pubkey string // base64 form, as stored in Settings.MinisignPubKey
}

func newTestSigner(t *testing.T) testSigner {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var sk minisign.PrivateKey
	copy(sk.SignatureAlgorithm[:], "Ed")
	if _, err := rand.Read(sk.KeyId[:]); err != nil {
		t.Fatal(err)
	}
	copy(sk.SecretKey[:], priv)

	raw := append(append([]byte("Ed"), sk.KeyId[:]...), pub...)
	return testSigner{sk: sk, pubkey: base64.StdEncoding.EncodeToString(raw)}
}

// sign returns a .minisig for data with the given trusted comment (without
// the "trusted comment: " prefix), in the prehashed mode Zig uses.
func (s testSigner) sign(t *testing.T, data []byte, trusted string) []byte {
	t.Helper()
	sig, err := s.sk.Sign(data, minisign.SignOptions{TrustedComment: trusted, Hashed: true})
	if err != nil {
		t.Fatal(err)
	}
	return sig.Encode()
}

func (s testSigner) signFile(t *testing.T, data []byte, name string) []byte {
	return s.sign(t, data, "timestamp:1760215991\tfile:"+name+"\thashed")
}

func (s testSigner) publicKey(t *testing.T) *minisign.PublicKey {
	t.Helper()
	pk, err := minisign.NewPublicKey(s.pubkey)
	if err != nil {
		t.Fatal(err)
	}
	return &pk
}

// fileServer serves a fixed set of paths and counts requests per path.
type fileServer struct {
	*httptest.Server
	mu    sync.Mutex
	files map[string][]byte
	hits  map[string]int
}

func newFileServer(t *testing.T, files map[string][]byte) *fileServer {
	t.Helper()
	fs := &fileServer{files: files, hits: map[string]int{}}
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.mu.Lock()
		fs.hits[r.URL.Path]++
		body, ok := fs.files[r.URL.Path]
		fs.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(fs.Close)
	return fs
}

func (fs *fileServer) hitCount(path string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.hits[path]
}

func stubShuffle(t *testing.T) {
	t.Helper()
	orig := shuffleMirrors
	shuffleMirrors = func([]string) {}
	t.Cleanup(func() { shuffleMirrors = orig })
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func tempFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if e.Name() != knownHashesFile {
			names = append(names, e.Name())
		}
	}
	return names
}

func TestDownloadVerified(t *testing.T) {
	const name = "zig-x86_64-linux-0.15.2.tar.xz"
	const oldName = "zig-x86_64-linux-0.14.1.tar.xz"
	signer := newTestSigner(t)
	other := newTestSigner(t)

	good := []byte("genuine 0.15.2 tarball")
	old := []byte("genuine but older 0.14.1 tarball")
	goodSig := signer.signFile(t, good, name)

	// A genuine signature whose trusted comment was rewritten afterwards.
	editedSig := bytes.Replace(signer.signFile(t, good, oldName), []byte(oldName), []byte(name), 1)

	tests := []struct {
		name    string
		tarball []byte
		sig     []byte // nil: no .minisig served
		shasum  string
		pinned  string // pre-recorded hash for name
		wantErr error
	}{
		{name: "valid", tarball: good, sig: goodSig, shasum: sha256Hex(good)},
		{name: "valid without shasum", tarball: good, sig: goodSig},
		{name: "valid and pinned", tarball: good, sig: goodSig, pinned: sha256Hex(good)},
		{name: "tampered tarball", tarball: []byte("tampered"), sig: goodSig, wantErr: ErrMinisignVerify},
		{name: "tampered tarball with shasum", tarball: []byte("tampered"), sig: goodSig, shasum: sha256Hex(good), wantErr: ErrShasumMismatch},
		{name: "older release under new name", tarball: old, sig: signer.signFile(t, old, oldName), wantErr: ErrSignedFilenameMismatch},
		{name: "no file field", tarball: good, sig: signer.sign(t, good, "timestamp:1760215991"), wantErr: ErrMissingSignedFilename},
		{name: "trusted comment edited", tarball: good, sig: editedSig, wantErr: ErrMinisignVerify},
		{name: "signed by another key", tarball: good, sig: other.signFile(t, good, name), wantErr: ErrMinisignVerify},
		{name: "missing signature", tarball: good, wantErr: ErrMinisignVerify},
		{name: "same name, different signed tarball than pinned", tarball: good, sig: goodSig, pinned: sha256Hex(old), wantErr: ErrKnownHashMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string][]byte{"/" + name: tt.tarball}
			if tt.sig != nil {
				files["/"+name+".minisig"] = tt.sig
			}
			srv := newFileServer(t, files)
			dir := t.TempDir()
			pins := filepath.Join(dir, knownHashesFile)
			if tt.pinned != "" {
				if err := recordKnownHash(pins, name, tt.pinned); err != nil {
					t.Fatal(err)
				}
			}

			chk := tarballCheck{name: name, shasum: tt.shasum, pubkey: signer.publicKey(t), knownHashes: pins, label: "test"}
			got, err := downloadVerified(dir, srv.URL+"/"+name, chk)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("downloadVerified() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if left := tempFiles(t, dir); len(left) != 0 {
					t.Fatalf("rejected download left files behind: %v", left)
				}
				if tt.pinned == "" {
					if _, err := os.Stat(pins); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("rejected download was pinned")
					}
				}
				return
			}

			data, err := os.ReadFile(got)
			if err != nil || !bytes.Equal(data, good) {
				t.Fatalf("downloaded file = %q, %v; want %q", data, err, good)
			}
			if _, err := checkKnownHash(pins, name, sha256Hex(good)); err != nil {
				t.Fatalf("verified download not pinned: %v", err)
			}
		})
	}
}

// A custom (unsigned) source keeps working without a signature or pinning.
func TestDownloadVerifiedUnsignedSource(t *testing.T) {
	const name = "zig-x86_64-linux-0.15.2.tar.xz"
	body := []byte("custom build")
	srv := newFileServer(t, map[string][]byte{"/" + name: body})
	dir := t.TempDir()

	got, err := downloadVerified(dir, srv.URL+"/"+name, tarballCheck{name: name, shasum: sha256Hex(body), label: "test"})
	if err != nil {
		t.Fatalf("downloadVerified() error = %v", err)
	}
	os.Remove(got)
	if n := srv.hitCount("/" + name + ".minisig"); n != 0 {
		t.Fatalf("fetched a signature for an unsigned source (%d requests)", n)
	}
	if _, err := os.Stat(filepath.Join(dir, knownHashesFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unsigned source was pinned")
	}
}

// --- end-to-end Install through the mirror list --------------------------

// zigTarball builds a minimal official-looking release archive whose single
// top-level directory is named after the tarball, as Zig's are.
func zigTarball(t *testing.T, stem, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	xw, err := xz.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(xw)
	if err := tw.WriteHeader(&tar.Header{Name: stem + "/", Typeflag: tar.TypeDir, Mode: 0755}); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: stem + "/zig", Typeflag: tar.TypeReg, Mode: 0755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := xw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type installFixture struct {
	z                     *ZVM
	signer                testSigner
	name                  string // tarball file name for 0.15.2
	good, old             []byte
	origin, bad, goodMirr *fileServer
}

// newInstallFixture serves an official-looking index.json from origin (which
// stands in for ziglang.org), a mirror list naming bad then good, and points
// Settings at them. DefaultSettings.VersionMapUrl is swapped so the test index
// counts as the official source.
func newInstallFixture(t *testing.T, bad func(f *installFixture) map[string][]byte) *installFixture {
	t.Helper()
	if zigArchiveExtension() != ".tar.xz" {
		t.Skip("fixture builds .tar.xz archives")
	}
	stubShuffle(t)
	arch, osName := zigStyleSysInfo()
	f := &installFixture{signer: newTestSigner(t)}
	f.name = fmt.Sprintf("zig-%s-%s-0.15.2.tar.xz", arch, osName)
	oldName := fmt.Sprintf("zig-%s-%s-0.14.1.tar.xz", arch, osName)
	f.good = zigTarball(t, strings.TrimSuffix(f.name, ".tar.xz"), "zig 0.15.2")
	f.old = zigTarball(t, strings.TrimSuffix(oldName, ".tar.xz"), "zig 0.14.1")

	f.origin = newFileServer(t, map[string][]byte{})
	f.goodMirr = newFileServer(t, map[string][]byte{
		"/" + f.name:              f.good,
		"/" + f.name + ".minisig": f.signer.signFile(t, f.good, f.name),
	})
	f.bad = newFileServer(t, bad(f))

	index, _ := json.Marshal(map[string]any{
		"0.15.2": map[string]any{
			arch + "-" + osName: map[string]any{
				"tarball": f.origin.URL + "/download/0.15.2/" + f.name,
				"shasum":  sha256Hex(f.good),
			},
		},
	})
	f.origin.files["/index.json"] = index
	f.origin.files["/mirrors.txt"] = []byte(f.bad.URL + "\n" + f.goodMirr.URL + "\n")

	origDefault := DefaultSettings.VersionMapUrl
	DefaultSettings.VersionMapUrl = f.origin.URL + "/index.json"
	t.Cleanup(func() { DefaultSettings.VersionMapUrl = origDefault })

	settings := DefaultSettings
	settings.VersionMapUrl = DefaultSettings.VersionMapUrl
	settings.MirrorListUrl = f.origin.URL + "/mirrors.txt"
	settings.MinisignPubKey = f.signer.pubkey
	f.z = &ZVM{baseDir: t.TempDir(), Settings: settings}
	return f
}

func TestInstallFallsBackPastBadMirror(t *testing.T) {
	tests := []struct {
		name       string
		skipShasum bool
		bad        func(f *installFixture) map[string][]byte
	}{
		{
			name: "older release served under the requested name",
			bad: func(f *installFixture) map[string][]byte {
				arch, osName := zigStyleSysInfo()
				oldName := fmt.Sprintf("zig-%s-%s-0.14.1.tar.xz", arch, osName)
				return map[string][]byte{"/" + f.name: f.old, "/" + f.name + ".minisig": f.signer.signFile(t, f.old, oldName)}
			},
		},
		{
			name:       "older release served under the requested name, --skip-shasum",
			skipShasum: true,
			bad: func(f *installFixture) map[string][]byte {
				arch, osName := zigStyleSysInfo()
				oldName := fmt.Sprintf("zig-%s-%s-0.14.1.tar.xz", arch, osName)
				return map[string][]byte{"/" + f.name: f.old, "/" + f.name + ".minisig": f.signer.signFile(t, f.old, oldName)}
			},
		},
		{
			name:       "tampered tarball, --skip-shasum",
			skipShasum: true,
			bad: func(f *installFixture) map[string][]byte {
				return map[string][]byte{"/" + f.name: []byte("tampered"), "/" + f.name + ".minisig": f.signer.signFile(t, f.good, f.name)}
			},
		},
		{
			name: "tampered tarball",
			bad: func(f *installFixture) map[string][]byte {
				return map[string][]byte{"/" + f.name: []byte("tampered"), "/" + f.name + ".minisig": f.signer.signFile(t, f.good, f.name)}
			},
		},
		{
			name: "no signature",
			bad: func(f *installFixture) map[string][]byte {
				return map[string][]byte{"/" + f.name: f.good}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newInstallFixture(t, tt.bad)

			got, err := f.z.Install("0.15.2", false, tt.skipShasum, true, true)
			if err != nil {
				t.Fatalf("Install() error = %v", err)
			}
			if got != "0.15.2" {
				t.Fatalf("Install() version = %q", got)
			}

			if n := f.bad.hitCount("/" + f.name + ".minisig"); n != 1 {
				t.Fatalf("bad mirror signature requests = %d, want 1", n)
			}
			if n := f.goodMirr.hitCount("/" + f.name); n != 1 {
				t.Fatalf("good mirror tarball requests = %d, want 1", n)
			}
			if n := f.origin.hitCount("/download/0.15.2/" + f.name); n != 0 {
				t.Fatalf("fell through to origin (%d requests), want install from mirror 2", n)
			}

			zig, err := os.ReadFile(filepath.Join(f.z.baseDir, "0.15.2", "zig"))
			if err != nil || string(zig) != "zig 0.15.2" {
				t.Fatalf("installed zig = %q, %v; want %q", zig, err, "zig 0.15.2")
			}
			if left, _ := filepath.Glob(filepath.Join(f.z.baseDir, "*.xz")); len(left) != 0 {
				t.Fatalf("download archives left behind: %v", left)
			}
			if _, err := checkKnownHash(filepath.Join(f.z.baseDir, knownHashesFile), f.name, sha256Hex(f.good)); err != nil {
				t.Fatalf("installed tarball not pinned: %v", err)
			}
		})
	}
}

func TestInstallFailsWhenNoCandidateVerifies(t *testing.T) {
	f := newInstallFixture(t, func(f *installFixture) map[string][]byte { return map[string][]byte{} })
	delete(f.goodMirr.files, "/"+f.name+".minisig") // every candidate now lacks a valid signature

	_, err := f.z.Install("0.15.2", false, false, true, true)
	if !errors.Is(err, ErrDownloadFail) || !errors.Is(err, ErrMinisignVerify) {
		t.Fatalf("Install() error = %v, want %v and %v", err, ErrDownloadFail, ErrMinisignVerify)
	}
	if _, err := os.Stat(filepath.Join(f.z.baseDir, "0.15.2")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("version installed despite failed verification")
	}
}

func TestInstallRejectsIndexPointingAtOtherVersion(t *testing.T) {
	f := newInstallFixture(t, func(f *installFixture) map[string][]byte { return map[string][]byte{} })
	arch, osName := zigStyleSysInfo()
	oldName := fmt.Sprintf("zig-%s-%s-0.14.1.tar.xz", arch, osName)
	index, _ := json.Marshal(map[string]any{
		"0.15.2": map[string]any{arch + "-" + osName: map[string]any{
			"tarball": f.origin.URL + "/download/0.14.1/" + oldName,
			"shasum":  sha256Hex(f.old),
		}},
	})
	f.origin.files["/index.json"] = index

	_, err := f.z.Install("0.15.2", false, true, true, true)
	if !errors.Is(err, ErrTarballVersionMismatch) {
		t.Fatalf("Install() error = %v, want %v", err, ErrTarballVersionMismatch)
	}
}

// Once a tarball name is pinned, a different (even correctly signed) file
// under that name is refused from every source, including with --force.
func TestInstallPinnedHashMismatch(t *testing.T) {
	f := newInstallFixture(t, func(f *installFixture) map[string][]byte { return map[string][]byte{} })
	if _, err := f.z.Install("0.15.2", false, false, true, true); err != nil {
		t.Fatalf("first Install() error = %v", err)
	}

	// The publisher key now signs a different file under the same name, and
	// the index agrees with it: only the pin can tell.
	arch, osName := zigStyleSysInfo()
	other := zigTarball(t, strings.TrimSuffix(f.name, ".tar.xz"), "substituted")
	f.goodMirr.files["/"+f.name] = other
	f.goodMirr.files["/"+f.name+".minisig"] = f.signer.signFile(t, other, f.name)
	f.origin.files["/download/0.15.2/"+f.name] = other
	f.origin.files["/download/0.15.2/"+f.name+".minisig"] = f.signer.signFile(t, other, f.name)
	index, _ := json.Marshal(map[string]any{
		"0.15.2": map[string]any{arch + "-" + osName: map[string]any{
			"tarball": f.origin.URL + "/download/0.15.2/" + f.name,
			"shasum":  sha256Hex(other),
		}},
	})
	f.origin.files["/index.json"] = index

	_, err := f.z.Install("0.15.2", true, false, true, true)
	if !errors.Is(err, ErrKnownHashMismatch) {
		t.Fatalf("Install() error = %v, want %v", err, ErrKnownHashMismatch)
	}
	zig, _ := os.ReadFile(filepath.Join(f.z.baseDir, "0.15.2", "zig"))
	if string(zig) != "zig 0.15.2" {
		t.Fatalf("installed zig = %q, want the original", zig)
	}
}
