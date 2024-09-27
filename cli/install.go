// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/tristanisham/zvm/cli/meta"

	"github.com/charmbracelet/log"

	"github.com/tristanisham/clr"
)

func (z *ZVM) Install(version string, force bool) error {
	os.Mkdir(z.baseDir, 0755)
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

	tarPath, err := getTarPath(version, &rawVersionStructure)
	if err != nil {
		if errors.Is(err, ErrUnsupportedVersion) {
			return fmt.Errorf("%s: %q", err, version)
		} else {
			return err
		}
	}

	log.Debug("tarPath", "url", tarPath)

	tarResp, err := reqZigDownload(tarPath)
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
		clr_opt_ver_str = clr.Green(version)
	} else {
		clr_opt_ver_str = version
	}

	pbar := progressbar.DefaultBytes(
		int64(tarResp.ContentLength),
		fmt.Sprintf("Downloading %s:", clr_opt_ver_str),
	)

	hash := sha256.New()
	_, err = io.Copy(io.MultiWriter(tempDir, pbar, hash), tarResp.Body)
	if err != nil {
		return err
	}

	var shasum string

	shasum, err = getVersionShasum(version, &rawVersionStructure)
	if err != nil {
		return err
	}

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
		log.Warnf("No shasum provided by host")
	}

	// The base directory where all Zig files for the appropriate version are installed
	// installedVersionPath := filepath.Join(z.zvmBaseDir, version)
	fmt.Println("Extracting bundle...")

	if err := ExtractBundle(tempDir.Name(), z.baseDir); err != nil {
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

// reqZigDownload HTTP requests Zig downloads from the official site and mirrors
func reqZigDownload(tarURL string) (*http.Response, error) {
	log.Debug("requestWithMirror", "tarURL", tarURL)

	tarResp, err := attemptDownload(tarURL)
	if err != nil {
		return nil, err
	}

	if tarResp.StatusCode == 200 {
		return tarResp, nil
	}

	mirrors := []func(string) (string, error){mirrorHryx, mirrorMachEngine}

	for i, mirror := range mirrors {
		log.Debugf("requestWithMirror url #%d", i)

		newURL, err := mirror(tarURL)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrDownloadFail, err)
		}

		log.Debug(fmt.Sprintf("mirror %d", i), "url", newURL)

		tarResp, err = attemptDownload(newURL)
		if err != nil {
			log.Debug("mirror req err", "mirror", newURL, "error", err)
			continue
		}

		if tarResp.StatusCode == 200 {
			return tarResp, nil
		}
	}

	return nil, errors.Join(err, fmt.Errorf("all download attempts failed"))
}

func attemptDownload(url string) (*http.Response, error) {
	req, err := createDownloadReq(url)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	log.Debug("requestWithMirror", "status code", resp.StatusCode)

	return resp, nil
}

func createDownloadReq(tarURL string) (*http.Request, error) {
	zigArch, zigOS := zigStyleSysInfo()

	zigDownloadReq, err := http.NewRequest("GET", tarURL, nil)
	if err != nil {
		return nil, err
	}

	zigDownloadReq.Header.Set("User-Agent", "zvm "+meta.VERSION)
	zigDownloadReq.Header.Set("X-Client-Os", zigOS)
	zigDownloadReq.Header.Set("X-Client-Arch", zigArch)

	return zigDownloadReq, nil
}

// mirrorHryx returns the Hryx mirror url equivilant for a Zig Build tarball URL.
func mirrorHryx(url string) (string, error) {
	if !strings.HasPrefix(url, "https://ziglang.org/builds/") {
		return "", fmt.Errorf("%w: expected a url that started with https://ziglang.org/builds/. Recieved %q", ErrInvalidInput, url)
	}

	return strings.Replace(url, "https://ziglang.org/builds/", "https://zigmirror.hryx.net/zig/", 1), nil
}

// mirrorMachEngine returns the Mach Engine mirror url equivilant for a Zig Build tarball URL.
func mirrorMachEngine(url string) (string, error) {
	if !strings.HasPrefix(url, "https://ziglang.org/builds/") {
		return "", fmt.Errorf("%w: expected a url that started with https://ziglang.org/builds/. Recieved %q", ErrInvalidInput, url)
	}

	return strings.Replace(url, "https://ziglang.org/builds/", "https://pkg.machengine.org/zig/", 1), nil
}

type githubTaggedReleaseResponse struct {
	Assets []gitHubAsset // json array of platform binaries
}

type gitHubAsset struct {
	Url                string // url for asset json object
	Name               string // contains platform information about binary
	BrowserDownloadUrl string `json:"browser_download_url"` // download url
}

type zlsCIDownloadIndexResponse struct {
	Versions     map[string]zlsCIZLSVersion
	Latest       string // most recent ZLS version
	LatestTagged string // most recent tagged ZLS version
}

type zlsCIZLSVersion struct {
	ZLSVersion string
	Targets    []string
}

func getZLSDownloadUrl(version string, archDouble string) (string, string, error) {
	if version == "master" {
		resp, err := http.Get("https://zigtools-releases.nyc3.digitaloceanspaces.com/zls/index.json")
		if err != nil {
			return "", "", err
		}
		defer resp.Body.Close()

		var releaseBuffer bytes.Buffer
		_, err = releaseBuffer.ReadFrom(resp.Body)
		if err != nil {
			return "", "", err
		}

		var ciIndex zlsCIDownloadIndexResponse
		if err := json.Unmarshal(releaseBuffer.Bytes(), &ciIndex); err != nil {
			return "", "", err
		}

		exeName := "zls"
		if strings.Contains(archDouble, "windows") {
			exeName = "zls.exe"
		}

		format_url := "https://zigtools-releases.nyc3.digitaloceanspaces.com/zls/%v/%v/%v"
		return fmt.Sprintf(format_url, ciIndex.Latest, archDouble, exeName), ciIndex.Latest, nil
	} else {
		url := fmt.Sprintf("https://api.github.com/repos/zigtools/zls/releases/tags/%v", version)

		// get release information
		resp, err := http.Get(url)
		if err != nil {
			return "", "", err
		}
		defer resp.Body.Close()

		var releaseBuffer bytes.Buffer
		_, err = releaseBuffer.ReadFrom(resp.Body)
		if err != nil {
			return "", "", err
		}

		// getting list of assets
		var taggedReleaseResponse githubTaggedReleaseResponse
		if err := json.Unmarshal(releaseBuffer.Bytes(), &taggedReleaseResponse); err != nil {
			return "", "", err
		}

		if len(taggedReleaseResponse.Assets) == 0 {
			return "", "", errors.New("invalid ZLS version")
		}

		// getting platform information
		var downloadUrl string
		for _, asset := range taggedReleaseResponse.Assets {
			if strings.Contains(asset.Name, archDouble) {
				downloadUrl = asset.BrowserDownloadUrl
				break
			}
		}

		if downloadUrl == "" {
			return "", "", errors.New("invalid ZLS release URL")
		}

		return downloadUrl, version, nil
	}
}

func (z *ZVM) InstallZls(version string, force bool) error {
	if version != "master" && strings.Count(version, ".") != 2 {
		return fmt.Errorf("%w: versions are SEMVER (MAJOR.MINOR.MINUSCULE)", ErrUnsupportedVersion)
	}

	fmt.Println("Finding ZLS executable...")

	// make sure dir exists
	installDir := filepath.Join(z.baseDir, version)
	err := os.MkdirAll(installDir, 0755)
	if err != nil {
		return err
	}

	arch, osType := zigStyleSysInfo()
	expectedArchOs := fmt.Sprintf("%v-%v", arch, osType)

	filename := "zls"
	if osType == "windows" {
		filename += ".exe"
	}

	// master does not need unzipping, zpm just serves full binary
	shouldUnzip := version != "master"

	downloadUrl, selectedVersion, err := getZLSDownloadUrl(version, expectedArchOs)
	if err != nil {
		return err
	}

	if !force {
		installedVersion := ""
		targetZls := strings.TrimSpace(filepath.Join(z.baseDir, version, "zls"))
		if _, err := os.Stat(targetZls); err == nil {
			cmd := exec.Command(targetZls, "--version")
			var zigVersion strings.Builder
			cmd.Stdout = &zigVersion
			err := cmd.Run()
			if err != nil {
				log.Warn(err)
			}

			installedVersion = strings.TrimSpace(zigVersion.String())
		}
		if installedVersion == selectedVersion {
			fmt.Printf("ZLS version %s is already installed\n", installedVersion)
			return nil
		}
	}

	request, err := http.NewRequest("GET", downloadUrl, nil)
	if err != nil {
		return err
	}

	request.Header.Set("User-Agent", "zvm "+meta.VERSION)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// if resp.ContentLength == 0 {
	// 	return fmt.Errorf("invalid ZLS content length (%d bytes)", resp.ContentLength)
	// }

	pbar := progressbar.DefaultBytes(
		int64(response.ContentLength),
		"Downloading ZLS",
	)

	versionPath := filepath.Join(z.baseDir, version)
	binaryLocation := filepath.Join(versionPath, filename)

	if !shouldUnzip {
		file, err := os.Create(binaryLocation)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(io.MultiWriter(pbar, file), response.Body); err != nil {
			return err
		}
	} else {
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

		if _, err := io.Copy(io.MultiWriter(pbar, tempFile), response.Body); err != nil {
			return err
		}

		zlsTempDir, err := os.MkdirTemp(z.baseDir, "zls-*")
		if err != nil {
			return err
		}

		defer os.RemoveAll(zlsTempDir)

		fmt.Println("Extracting ZLS...") // Edgy bit
		if err := ExtractBundle(tempFile.Name(), zlsTempDir); err != nil {
			log.Fatal(err)
		}

		zlsPath, err := findZlsExecutable(zlsTempDir)
		if err != nil {
			return err
		}

		if err := os.Rename(zlsPath, filepath.Join(versionPath, filename)); err != nil {
			return err
		}

		if zlsPath == "" {
			return fmt.Errorf("could not find ZLS in %q", zlsTempDir)
		}

	}

	if err := os.Chmod(filepath.Join(versionPath, filename), 0755); err != nil {
		return err
	}

	z.createSymlink(version)
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
	if _, err := os.Lstat(filepath.Join(z.baseDir, "bin")); err == nil {
		fmt.Println("Removing old symlink")
		if err := os.RemoveAll(filepath.Join(z.baseDir, "bin")); err != nil {
			log.Fatal("could not remove bin", err)
		}

	}

	if err := meta.Symlink(filepath.Join(z.baseDir, version), filepath.Join(z.baseDir, "bin")); err != nil {
		log.Fatal(err)
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

	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {

		err := unzipFile(f, destination)
		if err != nil {
			meta.CtaFatal(err)
		}

	}

	return nil
}

func unzipFile(f *zip.File, destination string) error {
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil
}

type installRequest struct {
	Site, Package, Version string
}

func ExtractInstall(input string) installRequest {
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
