// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/jedisct1/go-minisign"
	"github.com/schollz/progressbar/v3"
)

const trustedCommentPrefix = "trusted comment: "

// signedFilename returns the `file:` field of a minisign trusted comment, as
// written by `minisign -S` (e.g. "trusted comment: timestamp:1760215991\tfile:zig-x86_64-linux-0.15.2.tar.xz\thashed").
// Fields are whitespace separated; unknown fields are ignored.
//
// The trusted comment is only authenticated once PublicKey.Verify succeeds,
// which checks the global signature over it.
func signedFilename(trustedComment string) (string, error) {
	comment, ok := strings.CutPrefix(trustedComment, trustedCommentPrefix)
	if !ok {
		return "", fmt.Errorf("%w: malformed trusted comment %q", ErrMissingSignedFilename, trustedComment)
	}

	var name string
	found := false
	for _, field := range strings.Fields(comment) {
		value, ok := strings.CutPrefix(field, "file:")
		if !ok {
			continue
		}
		if found {
			return "", fmt.Errorf("%w: trusted comment has more than one file field: %q", ErrMissingSignedFilename, comment)
		}
		name, found = value, true
	}

	if name == "" {
		return "", fmt.Errorf("%w: %q", ErrMissingSignedFilename, comment)
	}
	return name, nil
}

// checkSignedFilename reports whether sig was made for the file named want.
// minisign itself does not check this, so without it any correctly signed
// tarball (e.g. an older release) could be substituted for the requested one.
func checkSignedFilename(sig minisign.Signature, want string) error {
	got, err := signedFilename(sig.TrustedComment)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%w: signed for %q, requested %q", ErrSignedFilenameMismatch, got, want)
	}
	return nil
}

// tarballMatchesVersion reports whether an official tarball name such as
// "zig-x86_64-linux-0.15.2.tar.xz" (or the older "zig-linux-x86_64-0.11.0.tar.xz")
// is the build of version. The signed filename only proves which file was
// signed, so this binds that file to the version the user asked for.
func tarballMatchesVersion(tarName, version string) bool {
	stem := strings.TrimSuffix(strings.TrimSuffix(tarName, ".tar.xz"), ".zip")
	return version != "" && strings.HasPrefix(stem, "zig-") && strings.HasSuffix(stem, "-"+version)
}

// tarballCheck describes how a downloaded tarball must be verified before it is accepted.
type tarballCheck struct {
	name        string              // expected tarball basename
	shasum      string              // expected SHA-256 hex; empty skips the check
	pubkey      *minisign.PublicKey // nil for sources that are not minisign-signed
	knownHashes string              // path to the TOFU store; empty disables pinning
	label       string              // progress bar label
}

// shuffleMirrors randomizes mirror order to spread load. Tests replace it.
var shuffleMirrors = func(mirrors []string) {
	rand.Shuffle(len(mirrors), func(i, j int) { mirrors[i], mirrors[j] = mirrors[j], mirrors[i] })
}

// mirrorCandidates returns the URLs to try for tarURL: the community mirrors
// in random order, then tarURL itself (ziglang.org) as the final fallback.
func mirrorCandidates(mirrorListURL, tarURL string) []string {
	tarURLParsed, err := url.Parse(tarURL)
	if err != nil {
		return []string{tarURL}
	}
	tarName := path.Base(tarURLParsed.Path)

	resp, err := attemptDownload(mirrorListURL, nil)
	if err != nil {
		log.Warn("could not fetch mirror list; downloading from ziglang.org", "error", err)
		return []string{tarURL}
	}
	defer resp.Body.Close()

	mirrorBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		log.Warn("could not read mirror list; downloading from ziglang.org", "error", err)
		return []string{tarURL}
	}

	var mirrors []string
	for line := range strings.Lines(string(mirrorBytes)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		mirrorTarURL, err := url.JoinPath(line, tarName)
		if err != nil {
			log.Debug("mirror path error", "mirror", line, "error", err)
			continue
		}
		mirrors = append(mirrors, mirrorTarURL)
	}
	shuffleMirrors(mirrors)

	return append(mirrors, tarURL)
}

// downloadFirstVerified tries each candidate URL in order and returns the path
// of a temporary file (in dir) holding the first download that passes every
// check in chk. Candidates that fail are discarded. The caller removes the
// returned file.
func downloadFirstVerified(dir string, candidates []string, chk tarballCheck) (string, error) {
	var errs []error
	for i, candidate := range candidates {
		tempPath, err := downloadVerified(dir, candidate, chk)
		if err == nil {
			return tempPath, nil
		}
		if len(candidates) == 1 {
			return "", err
		}

		log.Debug("download candidate rejected", "url", candidate, "error", err)
		host := candidate
		if u, perr := url.Parse(candidate); perr == nil {
			host = u.Host
		}
		if i < len(candidates)-1 {
			fmt.Printf("Download from %s failed: %v\nTrying next mirror...\n", host, err)
		}
		errs = append(errs, fmt.Errorf("%s: %w", host, err))
	}

	return "", errors.Join(append([]error{ErrDownloadFail, errors.New("all download attempts failed")}, errs...)...)
}

// downloadVerified downloads tarURL into a new temporary file in dir and
// verifies it against chk. On any failure the file is removed.
func downloadVerified(dir, tarURL string, chk tarballCheck) (tempPath string, err error) {
	var sig minisign.Signature
	if chk.pubkey != nil {
		// Fetch the signature first, and reject a wrong-file signature before
		// downloading the tarball. This early check is unauthenticated, so it
		// is only used to reject; PublicKey.Verify below authenticates it.
		sig, err = attemptMinisigDownload(tarURL, nil)
		if err != nil {
			return "", fmt.Errorf("%w: signature download failed: %w", ErrMinisignVerify, err)
		}
		if err := checkSignedFilename(sig, chk.name); err != nil {
			return "", err
		}
	}

	tarResp, err := attemptDownload(tarURL, nil)
	if err != nil {
		return "", err
	}
	defer tarResp.Body.Close()

	tempFile, err := os.CreateTemp(dir, "*"+zigArchiveExtension())
	if err != nil {
		return "", err
	}
	defer func() {
		tempFile.Close()
		if err != nil {
			os.Remove(tempFile.Name())
		}
	}()

	pbar := progressbar.DefaultBytes(tarResp.ContentLength, fmt.Sprintf("Downloading %s:", chk.label))
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tempFile, pbar, hash), tarResp.Body); err != nil {
		return "", fmt.Errorf("%w: %w", ErrDownloadFail, err)
	}
	if err := tempFile.Close(); err != nil {
		return "", err
	}
	sum := hex.EncodeToString(hash.Sum(nil))

	if chk.shasum != "" {
		fmt.Println("Checking shasum...")
		log.Debug("shasum check:", "theirs", chk.shasum, "ours", sum)
		if sum != chk.shasum {
			return "", fmt.Errorf("%w: %s", ErrShasumMismatch, chk.name)
		}
		fmt.Println("Shasums match! 🎉")
	}

	if chk.pubkey != nil {
		fmt.Println("Checking minisign signature...")
		// Verify checks both the signature over the file and the global
		// signature over the trusted comment, so the file: field checked
		// above is authenticated once this succeeds.
		ok, err := chk.pubkey.VerifyFromFile(tempFile.Name(), sig)
		if err != nil {
			return "", fmt.Errorf("%w: %s: %w", ErrMinisignVerify, chk.name, err)
		}
		if !ok {
			return "", fmt.Errorf("%w: %s", ErrMinisignVerify, chk.name)
		}
		fmt.Printf("Minisign signature verified for %s! 🎉\n", chk.name)
	}

	if chk.knownHashes != "" {
		recorded, err := checkKnownHash(chk.knownHashes, chk.name, sum)
		if err != nil {
			return "", err
		}
		if !recorded {
			if err := recordKnownHash(chk.knownHashes, chk.name, sum); err != nil {
				log.Warn("could not record tarball hash", "file", chk.knownHashes, "error", err)
			}
		}
	}

	return tempFile.Name(), nil
}
