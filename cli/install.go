// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"archive/zip"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/jedisct1/go-minisign"
	"github.com/schollz/progressbar/v3"
	"github.com/tristanisham/zvm/cli/meta"

	"github.com/charmbracelet/log"

	"github.com/tristanisham/clr"
)

// devVersionRegex is a pre-compiled regex pattern to match development versions
// Pattern: major.minor.patch-dev.number+commit (e.g. 0.16.0-dev.1334+06d08daba)
var devVersionRegex = regexp.MustCompile(`^\d+\.\d+\.\d+-dev\.\d+\+[0-9a-f]+$`)

// zigBuildsBaseURL is the base URL for Zig development builds
const zigBuildsBaseURL = "https://ziglang.org/builds/"

// IsDevelopmentVersion checks if the version string matches a development version pattern
func IsDevelopmentVersion(version string) bool {
	return devVersionRegex.MatchString(version)
}

// constructDevVersionURL builds the direct download URL for a development version
func constructDevVersionURL(version string) string {
	arch, osName := zigStyleSysInfo()
	// Development versions follow the pattern: {zigBuildsBaseURL}zig-{arch}-{os}-{version}.tar.xz
	url := fmt.Sprintf(zigBuildsBaseURL+"zig-%s-%s-%s.tar.xz", arch, osName, version)
	return url
}

func (z *ZVM) Install(version string, force bool, skipShasum bool, mirror bool) error {
	err := os.MkdirAll(z.baseDir, 0755)
	if err != nil {
		return err
	}

	// Check if this is a development version
	isDevVersion := IsDevelopmentVersion(version)
	var tarPath string
	var shasum string

	if isDevVersion {
		// For development versions, construct URL directly
		tarPath = constructDevVersionURL(version)
		log.Debug("Development version detected, using direct URL", "version", version, "url", tarPath)
	} else {
		// For regular versions, use version map
		rawVersionStructure, err := z.fetchVersionMap()
		if err != nil {
			return err
		}

		if !force {
			installedVersions, err := z.GetInstalledVersions()
			if err != nil {
				return err
			}
			if slices.Contains(installedVersions, version) {
				alreadyInstalled := true
				installedVersion := version
				if version == "master" {
					targetZig := strings.TrimSpace(filepath.Join(z.baseDir, "master", "zig"))
					cmd := exec.Command(targetZig, "version")
					var zigVersion strings.Builder
					cmd.Stdout = &zigVersion
					err := cmd.Run()
					if err != nil {
						log.Warn(err)
					}

					installedVersion = strings.TrimSpace(zigVersion.String())
					if master, ok := rawVersionStructure["master"]; ok {
						if remoteVersion, ok := master["version"].(string); ok {
							if installedVersion != remoteVersion {
								alreadyInstalled = false
							}
						}
					}
				}
				if alreadyInstalled {
					fmt.Printf("Zig version %s is already installed\nRerun with the `--force` flag to install anyway\n", installedVersion)
					return nil
				}
			}
		}

		tarPath, err = getTarPath(version, &rawVersionStructure)
		if err != nil {
			if errors.Is(err, ErrUnsupportedVersion) {
				return fmt.Errorf("%s: %q", err, version)
			} else {
				return err
			}
		}

		// Get shasum for regular versions (development versions don't have shasums in version map)
		shasum, err = getVersionShasum(version, &rawVersionStructure)
		if err != nil {
			return err
		}
	}

	log.Debug("tarPath", "url", tarPath)

	var tarResp *http.Response
	var minisig minisign.Signature

	if isDevVersion {
		// Development versions typically don't use mirrors
		mirror = false
	} else {
		mirror = mirror && z.Settings.UseMirrorList() && z.Settings.VersionMapUrl == DefaultSettings.VersionMapUrl
	}

	if mirror {
		tarResp, minisig, err = attemptMirrorDownload(z.Settings.MirrorListUrl, tarPath)
	} else {
		tarResp, err = attemptDownload(tarPath)
	}

	if err != nil {
		return err
	}
	defer tarResp.Body.Close()

	var pathEnding string
	if runtime.GOOS == "windows" {
		pathEnding = "*.zip"
	} else {
		pathEnding = "*.tar.xz"
	}

	tempFile, err := os.CreateTemp(z.baseDir, pathEnding)
	if err != nil {
		return err
	}

	defer tempFile.Close()
	defer os.RemoveAll(tempFile.Name())

	var clrOptVerStr string
	if z.Settings.UseColor {
		clrOptVerStr = clr.Green(version)
	} else {
		clrOptVerStr = version
	}

	pbar := progressbar.DefaultBytes(
		int64(tarResp.ContentLength),
		fmt.Sprintf("Downloading %s:", clrOptVerStr),
	)

	hash := sha256.New()
	_, err = io.Copy(io.MultiWriter(tempFile, pbar, hash), tarResp.Body)
	if err != nil {
		return err
	}

	if !skipShasum {
		fmt.Println("Checking shasum...")
		if len(shasum) > 0 {
			ourHexHash := hex.EncodeToString(hash.Sum(nil))
			log.Debug("shasum check:", "theirs", shasum, "ours", ourHexHash)
			if ourHexHash != shasum {
				// TODO (tristan)
				// Why is my sha256 identical on the server and sha256sum,
				// but not when I download it in ZVM? Oh shit.
				// It's because it's a compressed download.
				return fmt.Errorf("shasum for %v does not match expected value", version)
			}
			fmt.Println("Shasums match! 🎉")
		} else {
			if isDevVersion {
				log.Warnf("Dev versions don't have shasum, it's recommended to install it with --skip-shasum")
			} else {
				log.Warnf("No shasum provided by host")
			}
		}
	} else {
		fmt.Println("Skipping shasum check (user requested)")
	}

	if mirror {
		fmt.Println("Checking minisign signature...")
		pubkey, err := minisign.NewPublicKey(z.Settings.MinisignPubKey)
		if err != nil {
			return fmt.Errorf("minisign public key decoding failed: %v", err)
		}
		verified, err := pubkey.VerifyFromFile(tempFile.Name(), minisig)
		if err != nil {
			return fmt.Errorf("minisign verification failed: %v", err)
		}

		if !verified {
			return fmt.Errorf("minisign signature for %v could not be verified", version)
		}

		fmt.Println("Minisign signature verified! 🎉")
	}

	// The base directory where all Zig files for the appropriate version are installed
	// installedVersionPath := filepath.Join(z.zvmBaseDir, version)
	fmt.Println("Extracting bundle...")

	if err := ExtractBundle(tempFile.Name(), z.baseDir); err != nil {
		log.Fatal(err)
	}
	var tarName string

	resultUrl, err := url.Parse(tarPath)
	if err != nil {
		log.Error(err)
		tarName = version
	}

	// Maybe think of a better algorithm
	urlPath := strings.Split(resultUrl.Path, "/")
	tarName = urlPath[len(urlPath)-1]
	tarName = strings.TrimSuffix(tarName, ".tar.xz")
	tarName = strings.TrimSuffix(tarName, ".zip")

	if err := os.Rename(filepath.Join(z.baseDir, tarName), filepath.Join(z.baseDir, version)); err != nil {
		if _, err := os.Stat(filepath.Join(z.baseDir, version)); err == nil {
			// Room here to make the backup file.
			log.Debug("removing", "path", filepath.Join(z.baseDir, version))
			if err := os.RemoveAll(filepath.Join(z.baseDir, version)); err != nil {
				log.Fatal(err)
			} else {
				oldName := filepath.Join(z.baseDir, tarName)
				newName := filepath.Join(z.baseDir, version)
				log.Debug("renaming", "old", oldName, "new", newName, "identical", oldName == newName)
				if oldName != newName {
					if err := os.Rename(oldName, newName); err != nil {
						log.Fatal(clr.Yellow(err))
					}
				}

			}

		}
	}

	// This removes the extra download
	if err := os.RemoveAll(filepath.Join(z.baseDir, tarName)); err != nil {
		log.Warn(err)
	}

	z.createSymlink(version)

	fmt.Println("Successfully installed Zig!")

	return nil
}

// attemptMirrorDownload HTTP requests Zig downloads from the community mirrorlist.
// Returns a tuple of (response, minisig, error).
func attemptMirrorDownload(mirrorListURL string, tarURL string) (*http.Response, minisign.Signature, error) {
	log.Debug("attemptMirrorDownload", "mirrorListURL", mirrorListURL, "tarURL", tarURL)
	tarURLParsed, err := url.Parse(tarURL)
	if err != nil {
		return nil, minisign.Signature{}, fmt.Errorf("%w: %w", ErrDownloadFail, err)
	}
	tarName := path.Base(tarURLParsed.Path)

	resp, err := attemptDownload(mirrorListURL)
	if err != nil {
		return nil, minisign.Signature{}, fmt.Errorf("%w: %w", ErrDownloadFail, err)
	}
	defer resp.Body.Close()

	mirrorBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, minisign.Signature{}, err
	}

	mirrors := strings.Split(string(mirrorBytes), "\n")
	// Pop empty field after terminating newline
	mirrors = mirrors[:len(mirrors)-1]
	rand.Shuffle(len(mirrors), func(i, j int) { mirrors[i], mirrors[j] = mirrors[j], mirrors[i] })
	// Default as fallback
	mirrors = append(mirrors, zigBuildsBaseURL)

	for i, mirror := range mirrors {
		mirrorTarURL, err := url.JoinPath(mirror, tarName)
		if err != nil {
			log.Debug("mirror path error", "mirror", mirror, "error", err)
			continue
		}

		log.Debug("attemptMirrorDownload", "mirror", i, "mirrorURL", mirrorTarURL)
		tarResp, err := attemptDownload(mirrorTarURL)
		if err != nil {
			log.Debug("mirror tar error", "mirror", mirror, "error", err)
			continue
		}

		minisig, err := attemptMinisigDownload(mirrorTarURL)
		if err != nil {
			log.Debug("mirror minisig error", "mirror", mirror, "error", err)
			tarResp.Body.Close()
			continue
		}

		return tarResp, minisig, nil
	}

	return nil, minisign.Signature{}, fmt.Errorf("%w: %w: %w", ErrDownloadFail, errors.New("all download attempts failed"), err)
}

func attemptMinisigDownload(tarURL string) (minisign.Signature, error) {
	minisigResp, err := attemptDownload(tarURL + ".minisig")
	if err != nil {
		return minisign.Signature{}, err
	}
	defer minisigResp.Body.Close()

	minisigBytes, err := io.ReadAll(minisigResp.Body)
	if err != nil {
		return minisign.Signature{}, err
	}

	return minisign.DecodeSignature(string(minisigBytes))
}

// attemptDownload creates a generic http request for ZVM.
func attemptDownload(url string) (*http.Response, error) {
	req, err := createDownloadReq(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDownloadFail, err)
	}

	client := http.DefaultClient

	// Checks the ZVM_SKIP_TLS_VERIFY environment variable and
	// toggles verifying a secure connection.
	if kind, is := os.LookupEnv("ZVM_SKIP_TLS_VERIFY"); is {

		if kind != "no-warn" {
			log.Warnf("ZVM_SKIP_TLS_VERIFY enabled")
		}

		log.Debug("ZVM_SKIP_TLS_VERIFY", "enabled", true)
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	} else {
		// Yeah, yeah. Just an easy way to do the call.
		log.Debug("ZVM_SKIP_TLS_VERIFY", "enabled", false)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDownloadFail, err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%w: %s", ErrDownloadFail, resp.Status)
	}

	return resp, nil
}

func createDownloadReq(tarURL string) (*http.Request, error) {
	zigArch, zigOS := zigStyleSysInfo()

	zigDownloadReq, err := http.NewRequest("GET", tarURL+"?source=zvm", nil)
	if err != nil {
		return nil, err
	}

	zigDownloadReq.Header.Set("User-Agent", "zvm "+meta.VERSION)
	zigDownloadReq.Header.Set("X-Client-Os", zigOS)
	zigDownloadReq.Header.Set("X-Client-Arch", zigArch)

	return zigDownloadReq, nil
}

func (z *ZVM) SelectZlsVersion(version string, compatMode string) (string, string, string, error) {
	rawVersionStructure, err := z.fetchZlsTaggedVersionMap()
	if err != nil {
		return "", "", "", err
	}

	// tagged releases.
	tarPath, err := getTarPath(version, &rawVersionStructure)
	if err == nil {
		shasum, err := getVersionShasum(version, &rawVersionStructure)
		if err == nil {
			return version, tarPath, shasum, nil
		}
	}

	// master/nightly releases.
	if err == ErrUnsupportedVersion {
		info, err := z.fetchZlsVersionByZigVersion(version, compatMode)
		if err != nil {
			return "", "", "", err
		}

		zlsVersion, ok := info["version"].(string)
		if !ok {
			return "", "", "", ErrMissingVersionInfo
		}

		arch, ops := zigStyleSysInfo()
		systemInfo, ok := info[fmt.Sprintf("%s-%s", arch, ops)].(map[string]any)
		if !ok {
			return "", "", "", ErrUnsupportedSystem
		}

		tar, ok := systemInfo["tarball"].(string)
		if !ok {
			return "", "", "", ErrMissingBundlePath
		}

		shasum, ok := systemInfo["shasum"].(string)
		if !ok {
			return "", "", "", ErrMissingShasum
		}

		return zlsVersion, tar, shasum, nil
	}

	return "", "", "", err
}

func (z *ZVM) InstallZls(requestedVersion string, compatMode string, force bool, skipShasum bool) error {
	fmt.Println("Determining installed Zig version...")

	// make sure dir exists
	installDir := filepath.Join(z.baseDir, requestedVersion)
	err := os.MkdirAll(installDir, 0755)
	if err != nil {
		return err
	}

	targetZig := strings.TrimSpace(filepath.Join(z.baseDir, requestedVersion, "zig"))
	cmd := exec.Command(targetZig, "version")
	var builder strings.Builder
	cmd.Stdout = &builder
	err = cmd.Run()
	if err != nil {
		log.Warn(err)
	}
	zigVersion := strings.TrimSpace(builder.String())
	log.Debug("installed zig version", "version", zigVersion)

	fmt.Println("Selecting ZLS version...")

	zlsVersion, tarPath, shasum, err := z.SelectZlsVersion(zigVersion, compatMode)
	if err != nil {
		if errors.Is(err, ErrUnsupportedVersion) {
			return fmt.Errorf("%s: %q", err, zigVersion)
		} else {
			return err
		}
	}
	log.Debug("selected zls version", "zigVersion", zigVersion, "zlsVersion", zlsVersion)

	_, osType := zigStyleSysInfo()
	filename := "zls"
	if osType == "windows" {
		filename += ".exe"
	}

	if !force {
		installedVersion := ""
		targetZls := strings.TrimSpace(filepath.Join(installDir, filename))
		if _, err := os.Stat(targetZls); err == nil {
			cmd := exec.Command(targetZls, "--version")
			var builder strings.Builder
			cmd.Stdout = &builder
			err := cmd.Run()
			if err != nil {
				log.Warn(err)
			}

			installedVersion = strings.TrimSpace(builder.String())
		}
		if installedVersion == zlsVersion {
			fmt.Printf("ZLS version %s is already installed\n", installedVersion)
			return nil
		}
	}

	log.Debug("tarPath", "url", tarPath)

	tarResp, err := attemptDownload(tarPath)
	if err != nil {
		return err
	}
	defer tarResp.Body.Close()

	var pathEnding string
	if runtime.GOOS == "windows" {
		pathEnding = "*.zip"
	} else {
		pathEnding = "*.tar.xz"
	}

	tempDir, err := os.CreateTemp(z.baseDir, pathEnding)
	if err != nil {
		return err
	}

	defer tempDir.Close()
	defer os.RemoveAll(tempDir.Name())

	var clr_opt_ver_str string
	if z.Settings.UseColor {
		clr_opt_ver_str = clr.Green(zlsVersion)
	} else {
		clr_opt_ver_str = zlsVersion
	}

	pbar := progressbar.DefaultBytes(
		int64(tarResp.ContentLength),
		fmt.Sprintf("Downloading ZLS %s:", clr_opt_ver_str),
	)

	hash := sha256.New()
	_, err = io.Copy(io.MultiWriter(tempDir, pbar, hash), tarResp.Body)
	if err != nil {
		return err
	}

	if !skipShasum {
		fmt.Println("Checking ZLS shasum...")
		if len(shasum) > 0 {
			ourHexHash := hex.EncodeToString(hash.Sum(nil))
			log.Debug("shasum check:", "theirs", shasum, "ours", ourHexHash)
			if ourHexHash != shasum {
				// TODO (tristan)
				// Why is my sha256 identical on the server and sha256sum,
				// but not when I download it in ZVM? Oh shit.
				// It's because it's a compressed download.
				return fmt.Errorf("shasum for zls-%v does not match expected value", zlsVersion)
			}
			fmt.Println("Shasums for ZLS match! 🎉")
		} else {
			log.Warnf("No ZLS shasum provided by host")
		}
	} else {
		fmt.Println("Skipping ZLS shasum check (user requested)")
	}

	fmt.Println("Extracting ZLS bundle...")

	zlsTempDir, err := os.MkdirTemp(z.baseDir, "zls-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(zlsTempDir)

	if err := ExtractBundle(tempDir.Name(), zlsTempDir); err != nil {
		log.Fatal(err)
	}

	zlsPath, err := findZlsExecutable(zlsTempDir)
	if err != nil {
		return err
	}

	if err := os.Rename(zlsPath, filepath.Join(installDir, filename)); err != nil {
		return err
	}

	if zlsPath == "" {
		return fmt.Errorf("could not find ZLS in %q", zlsTempDir)
	}

	if err := os.Chmod(filepath.Join(installDir, filename), 0755); err != nil {
		return err
	}

	z.createSymlink(requestedVersion)
	fmt.Println("Done! 🎉")
	return nil
}

func findZlsExecutable(dir string) (string, error) {
	var result string

	filename := "zls"
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || d.Type().Type() == os.ModeSymlink {
			return nil
		}

		if filepath.Base(path) != filename {
			return nil
		}

		result = path

		return fs.SkipAll
	})
	if err != nil {
		return "", err
	}

	return result, nil
}

func (z *ZVM) createSymlink(version string) {
	// .zvm/master
	versionPath := filepath.Join(z.baseDir, version)
	binDir := filepath.Join(z.baseDir, "bin")

	stat, err := os.Lstat(binDir)

	// See zvm.Use() for an explanation.
	if stat != nil {
		if err == nil {
			fmt.Println("Removing old inode link")
			if err := os.RemoveAll(binDir); err != nil {
				log.Fatal("could not remove bin", "err", err, "dir", binDir)
			}

		}
	}

	if err := meta.Link(versionPath, binDir); err != nil {
		log.Fatal("meta.Link error", err)
	}

}

func getTarPath(version string, data *map[string]map[string]any) (string, error) {
	arch, ops := zigStyleSysInfo()

	if info, ok := (*data)[version]; ok {
		if systemInfo, ok := info[fmt.Sprintf("%s-%s", arch, ops)]; ok {
			if base, ok := systemInfo.(map[string]any); ok {
				if tar, ok := base["tarball"].(string); ok {
					return tar, nil
				}
			} else {
				return "", ErrMissingBundlePath
			}
		} else {
			return "", ErrUnsupportedSystem
		}
	}

	// verMap := []string{"  "}
	// for key := range *data {
	// 	verMap = append(verMap, key)
	// }

	// return nil, fmt.Errorf("invalid Zig version: %s\nAllowed versions:%s", version, strings.Join(verMap, "\n  "))

	return "", ErrUnsupportedVersion
}

func getVersionShasum(version string, data *map[string]map[string]any) (string, error) {
	if info, ok := (*data)[version]; ok {
		arch, ops := zigStyleSysInfo()
		if systemInfo, ok := info[fmt.Sprintf("%s-%s", arch, ops)]; ok {
			if base, ok := systemInfo.(map[string]any); ok {
				if shasum, ok := base["shasum"].(string); ok {
					return shasum, nil
				}
			} else {
				return "", fmt.Errorf("unable to find necessary download path")
			}
		} else {
			return "", fmt.Errorf("invalid/unsupported system: ARCH: %s OS: %s", arch, ops)
		}
	}
	verMap := []string{"  "}
	for key := range *data {
		verMap = append(verMap, key)
	}

	return "", fmt.Errorf("invalid Zig version: %s\nAllowed versions:%s", version, strings.Join(verMap, "\n  "))
}

func zigStyleSysInfo() (arch string, os string) {
	arch = runtime.GOARCH
	os = runtime.GOOS

	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	case "loong64":
		arch = "loongarch64"
	case "ppc64le":
		arch = "powerpc64le"
	}

	switch os {
	case "darwin":
		os = "macos"
	}

	return arch, os
}

func ExtractBundle(bundle, out string) error {
	// This is how I extracted an extension from a path in a cross-platform manner before
	// I realized filepath existed.
	// -----------------------------------------------------------------------------------
	// get extension
	// replacedBundle := strings.ReplaceAll(bundle, "\\", "/")
	// splitPath := strings.Split(replacedBundle, "/")
	// _, extension, _ := strings.Cut(splitPath[len(splitPath)-1], ".")
	extension := filepath.Ext(bundle)

	// For some reason, this broke inexplicably in v0.6.6. Added check for ".xz" extension
	// to fix, but would love to know how this became an issue.
	if strings.Contains(extension, "tar") || extension == ".xz" {
		return untarXZ(bundle, out)
	} else if extension == ".zip" {
		return unzipSource(bundle, out)
	}

	return fmt.Errorf("unknown format %v", extension)
}

func untarXZ(in, out string) error {
	tar := exec.Command("tar", "-xf", in, "-C", out)
	tar.Stdout = os.Stdout
	tar.Stderr = os.Stderr
	if err := tar.Run(); err != nil {
		log.Debug("Error untarring bundle")
		return err
	}
	return nil
}

func unzipSource(source, destination string) error {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}

	defer reader.Close()

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}

	os.MkdirAll(destination, 0755)

	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}

		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(destination, f.Name)
		if !strings.HasPrefix(path, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}

			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}

	}

	return nil
}

type installRequest struct {
	Site, Package, Version string
}

func ExtractInstall(input string) installRequest {
	log.Debug("ExtractInstall", "input", input)
	var req installRequest
	colonIdx := strings.Index(input, ":")
	atIdx := strings.Index(input, "@")

	if colonIdx != -1 {
		req.Site = input[:colonIdx]
		if atIdx != -1 {
			req.Package = input[colonIdx+1 : atIdx]
			req.Version = input[atIdx+1:]
		} else {
			req.Package = input[colonIdx+1:]
		}
	} else {
		if atIdx != -1 {
			req.Package = input[:atIdx]
			req.Version = input[atIdx+1:]
		} else {
			req.Package = input
		}
	}

	return req
}
