package cli

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/charmbracelet/log"

	"github.com/schollz/progressbar/v3"
	"github.com/tristanisham/clr"
)

func (z *ZVM) Install(version string) error {

	os.Mkdir(z.zvmBaseDir, 0755)

	rawVersionStructure, err := z.fetchVersionMap()
	if err != nil {
		return err
	}

	wasZigOnl := false
	tarPath, err := getTarPath(version, &rawVersionStructure)
	if err != nil {
		if errors.Is(err, ErrUnsupportedVersion) {
			tarPath, err = checkZigOnl(version)
			if err != nil {
				return err
			}
			wasZigOnl = true
		} else {
			return err

		}
	}

	tarReq, err := http.Get(tarPath)
	if err != nil {
		return err
	}
	defer tarReq.Body.Close()
	// _ = os.MkdirAll(filepath.Join(z.zvmBaseDir, version), 0755)
	// tarDownloadPath := filepath.Join(z.zvmBaseDir, version, fmt.Sprintf("%s.tar.xz", version))

	var pathEnding string
	if runtime.GOOS == "windows" {
		pathEnding = "*.zip"
	} else {
		pathEnding = "*.tar.xz"
	}

	tempDir, err := os.CreateTemp(z.zvmBaseDir, pathEnding)
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
		tarReq.ContentLength,
		fmt.Sprintf("Downloading %s:", clr_opt_ver_str),
	)

	hash := sha256.New()

	_, err = io.Copy(io.MultiWriter(tempDir, hash, pbar), tarReq.Body)
	if err != nil {
		return err
	}

	var shasum string
	if wasZigOnl {
		shasum = tarReq.Header.Get("X-Sha256")
	} else {
		shasum, err = getVersionShasum(version, &rawVersionStructure)
		if err != nil {
			return err
		}
	}

	fmt.Println("Checking shasum...")
	if len(shasum) > 0 {
		if hex.EncodeToString(hash.Sum(nil)) != shasum {
			return fmt.Errorf("shasum for %v does not match expected value", version)
		}
		fmt.Println("Shasums match! 🎉")
	} else {
		log.Warnf("No shasum. Downloaded from zig.onl: %v", wasZigOnl)
	}
	
	// The base directory where all Zig files for the appropriate version are installed
	// installedVersionPath := filepath.Join(z.zvmBaseDir, version)
	fmt.Println("Extracting bundle...")

	if err := ExtractBundle(tempDir.Name(), z.zvmBaseDir); err != nil {
		log.Fatal(err)
	}
	tarName := strings.TrimPrefix(tarPath, "https://ziglang.org/builds/")
	tarName = strings.TrimPrefix(tarName, fmt.Sprintf("https://ziglang.org/download/%s/", version))
	tarName = strings.TrimSuffix(tarName, ".tar.xz")
	tarName = strings.TrimSuffix(tarName, ".zip")

	// var finalInstallFolderName string = version

	// if version == "master" {

	// }

	if err := os.Rename(filepath.Join(z.zvmBaseDir, tarName), filepath.Join(z.zvmBaseDir, version)); err != nil {
		if _, err := os.Stat(filepath.Join(z.zvmBaseDir, version)); err == nil {
			// Room here to make the backup file.
			if err := os.RemoveAll(filepath.Join(z.zvmBaseDir, version)); err != nil {
				log.Fatal(err)
			} else {
				if err := os.Rename(filepath.Join(z.zvmBaseDir, tarName), filepath.Join(z.zvmBaseDir, version)); err != nil {
					log.Fatal(clr.Yellow(err))
				}
			}
		}
	}

	// This removes the extra download
	if err := os.RemoveAll(filepath.Join(z.zvmBaseDir, tarName)); err != nil {
		log.Warn(err)
	}

	if _, err := os.Lstat(filepath.Join(z.zvmBaseDir, "bin")); err == nil {
		os.Remove(filepath.Join(z.zvmBaseDir, "bin"))
	}

	if err := os.Symlink(filepath.Join(z.zvmBaseDir, version), filepath.Join(z.zvmBaseDir, "bin")); err != nil {
		log.Fatal(err)
	}

	return nil
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
	if runtime.GOOS == "windows" {
		return unzipSource(bundle, out)
	}
	return untarXZ(bundle, out)
}

func untarXZ(in, out string) error {
	tar := exec.Command("tar", "-xmf", in, "-C", out)
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

func checkZigOnl(version string) (string, error) {
	arch, os := zigStyleSysInfo()
	zigOnl := fmt.Sprintf("https://zig.onl/versions?release=%s&os=%s&arch=%s", version, os, arch)
	resp, err := http.Get(zigOnl)
	if err != nil {
		if err, ok := err.(*url.Error); ok {
			if err.Timeout() {
				return "", fmt.Errorf("timeout fetching %s", version)
			} else if err.Temporary() {
				return "", fmt.Errorf("temporary error fetching %s. Please try again", version)
			}
		}
	}

	if resp.StatusCode == 200 {
		return zigOnl, nil
	}

	if resp.StatusCode != 200 {
		return "", ErrUnsupportedVersion
	}
	return "", nil
}
