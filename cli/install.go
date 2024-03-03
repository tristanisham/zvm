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
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/tristanisham/zvm/cli/meta"

	"github.com/charmbracelet/log"

	"github.com/tristanisham/clr"
)

func (z *ZVM) Install(version string) error {

	os.Mkdir(z.baseDir, 0755)
	rawVersionStructure, err := z.fetchVersionMap()
	if err != nil {
		return err
	}

	wasZigOnl := false
	tarPath, err := getTarPath(version, &rawVersionStructure)
	if err != nil {
		if errors.Is(err, ErrUnsupportedVersion) {
			log.Fatalf("%s: %q", err, version)
		} else {
			return err
		}
	}

	zigArch, zigOS := zigStyleSysInfo()
	log.Debug("tarPath", "url", tarPath)
	zigDownloadReq, err := http.NewRequest("GET", tarPath, nil)
	if err != nil {
		return err
	}

	zigDownloadReq.Header.Set("User-Agent", "zvm "+meta.VERSION)
	zigDownloadReq.Header.Set("X-Client-Os", zigOS)
	zigDownloadReq.Header.Set("X-Client-Arch", zigArch)

	tarResp, err := http.DefaultClient.Do(zigDownloadReq)
	if err != nil {
		return err
	}
	defer tarResp.Body.Close()
	// _ = os.MkdirAll(filepath.Join(z.zvmBaseDir, version), 0755)
	// tarDownloadPath := filepath.Join(z.zvmBaseDir, version, fmt.Sprintf("%s.tar.xz", version))

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
		log.Warnf("No shasum. Downloaded from zig.onl: %v", wasZigOnl)
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
	if wasZigOnl {
		if rel := resultUrl.Query().Get("release"); len(rel) > 0 {
			tarName = strings.Replace(rel, " ", "+", 1)
		} else {
			tarName = version
		}

	} else {
		// Maybe think of a better algorithm
		urlPath := strings.Split(resultUrl.Path, "/")
		tarName = urlPath[len(urlPath)-1]
		tarName = strings.TrimSuffix(tarName, ".tar.xz")
		tarName = strings.TrimSuffix(tarName, ".zip")
	}

	if wasZigOnl {

		untarredPath := filepath.Join(z.baseDir, fmt.Sprintf("zig-%s-%s-%s", zigOS, zigArch, version))
		newPath := filepath.Join(z.baseDir, tarName)

		if _, err := os.Stat(untarredPath); err == nil {
			if os.Stat(newPath); err == nil {
				if err := os.RemoveAll(newPath); err == nil {
					if err := os.Rename(untarredPath, newPath); err != nil {
						log.Debug("rename err", "untarrPath", untarredPath, "newPath", newPath, "err", err)
						return fmt.Errorf("renaming error: rename %q to %q", untarredPath, newPath)
					}
				} else {
					log.Debug("remove existing install", "err", err)
					return err
				}

			}

		}
	} else {
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
	}

	z.createSymlink(version)

	return nil
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
	Latest       string // most recent ZLS version
	LatestTagged string // most recent tagged ZLS version
	Versions     map[string]zlsCIZLSVersion
}

type zlsCIZLSVersion struct {
	ZLSVersion string
	Targets    []string
}

func getZLSDownloadUrl(version string, archDouble string) (string, error) {
	if version == "master" {
		resp, err := http.Get("https://zigtools-releases.nyc3.digitaloceanspaces.com/zls/index.json")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var releaseBuffer bytes.Buffer
		_, err = releaseBuffer.ReadFrom(resp.Body)
		if err != nil {
			return "", err
		}

		var ciIndex zlsCIDownloadIndexResponse
		if err := json.Unmarshal(releaseBuffer.Bytes(), &ciIndex); err != nil {
			return "", err
		}

		exeName := "zls"
		if strings.Contains(archDouble, "windows") {
			exeName = "zls.exe"
		}

		format_url := "https://zigtools-releases.nyc3.digitaloceanspaces.com/zls/%v/%v/%v"
		return fmt.Sprintf(format_url, ciIndex.Latest, archDouble, exeName), nil
	} else {
		url := fmt.Sprintf("https://api.github.com/repos/zigtools/zls/releases/tags/%v", version)

		// get release information
		resp, err := http.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var releaseBuffer bytes.Buffer
		_, err = releaseBuffer.ReadFrom(resp.Body)
		if err != nil {
			return "", err
		}

		// getting list of assets
		var taggedReleaseResponse githubTaggedReleaseResponse
		if err := json.Unmarshal(releaseBuffer.Bytes(), &taggedReleaseResponse); err != nil {
			return "", err
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
			return "", errors.New("invalid release URl")
		}

		return downloadUrl, nil
	}
}

func (z *ZVM) InstallZls(version string) error {
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
	shouldUnzip := true
	if version == "master" {
		shouldUnzip = false
	}

	downloadUrl, err := getZLSDownloadUrl(version, expectedArchOs)
	if err != nil {
		return err
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

		tempDir, err := os.CreateTemp(z.baseDir, pathEnding)
		if err != nil {
			return err
		}

		defer tempDir.Close()
		defer os.RemoveAll(tempDir.Name())

		if _, err := io.Copy(io.MultiWriter(pbar, tempDir), response.Body); err != nil {
			return err
		}

		fmt.Println("Extracting ZLS...")
		if err := ExtractBundle(tempDir.Name(), filepath.Join(z.baseDir, version)); err != nil {
			log.Fatal(err)
		}
		if err := os.Rename(filepath.Join(versionPath, "bin", filename), filepath.Join(versionPath, filename)); err != nil {
			return err
		}
	}

	if err := os.Chmod(filepath.Join(versionPath, filename), 0755); err != nil {
		return err
	}

	z.createSymlink(version)
	fmt.Println("Done! 🎉")
	return nil
}

func (z *ZVM) createSymlink(version string) {
	if _, err := os.Lstat(filepath.Join(z.baseDir, "bin")); err == nil {
		fmt.Println("Removing old symlink")
		if err := os.RemoveAll(filepath.Join(z.baseDir, "bin")); err != nil {
			log.Fatal("could not remove bin", err)
		}

	}

	if runtime.GOOS == "windows" {
		elevatedRun("mklink", "/D", filepath.Join(z.baseDir, "bin"), filepath.Join(z.baseDir, version))
	} else {
		if err := os.Symlink(filepath.Join(z.baseDir, version), filepath.Join(z.baseDir, "bin")); err != nil {
			log.Fatal(err)
		}
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
	// get extension
	replacedBundle := strings.ReplaceAll(bundle, "\\", "/")
	splitPath := strings.Split(replacedBundle, "/")
	_, extension, _ := strings.Cut(splitPath[len(splitPath)-1], ".")

	if strings.Contains(extension, "tar") {
		return untarXZ(bundle, out)
	} else if strings.Contains(extension, "zip") {
		return unzipSource(bundle, out)
	}

	return fmt.Errorf("unknown format %v", extension)
}

func untarXZ(in, out string) error {
	tar := exec.Command("tar", "-xf", in, "-C", out)
	tar.Stdout = os.Stdout
	tar.Stderr = os.Stderr
	if err := tar.Run(); err != nil {
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
			return err
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

type InstallRequest struct {
	Site, Package, Version string
}

func ExtractInstall(input string) InstallRequest {
	var req InstallRequest
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
