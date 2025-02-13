// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/schollz/progressbar/v3"

	"github.com/tristanisham/zvm/cli/meta"

	"github.com/charmbracelet/log"
	"github.com/tristanisham/clr"
	"golang.org/x/mod/semver"
)

// Upgrade will upgrade the system installation of ZVM.
// I wrote most of it before I remembered that GitHub has an API so expect major refactoring.
func (z *ZVM) Upgrade() error {
	defer func() {
		if err := z.Clean(); err != nil {
			log.Warn("ZVM failed to clean up after itself.")
		}
	}()

	tagName, upgradable, err := CanIUpgrade()
	if err != nil {
		return errors.Join(ErrFailedUpgrade, err)
	}

	if !upgradable {
		fmt.Printf("You are already on the latest release (%s) of ZVM :) \n", clr.Blue(meta.VERSION))
		return nil
	} else {
		fmt.Printf("You are on ZVM %s... upgrading to (%s)", meta.VERSION, tagName)
	}

	zvmInstallDirENV, err := z.getInstallDir()
	if err != nil {
		return err
	}

	log.Debug("exe dir", "path", zvmInstallDirENV)
	if _, err := os.Stat(zvmInstallDirENV); errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(zvmInstallDirENV, 0775); err != nil {
			log.Fatal(err)
		}
	}
	zvmBinaryName := "zvm"
	archive := "tar"
	if runtime.GOOS == "windows" {
		archive = "zip"
		zvmBinaryName = "zvm.exe"
	}

	download := fmt.Sprintf("zvm-%s-%s.%s", runtime.GOOS, runtime.GOARCH, archive)

	downloadUrl := fmt.Sprintf("https://github.com/tristanisham/zvm/releases/latest/download/%s", download)
	log.Debugf("Downloading latest release from %s", downloadUrl)
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return errors.Join(ErrFailedUpgrade, err)
	}
	defer resp.Body.Close()
	log.Debugf("done")

	if err = os.MkdirAll(z.Directories.cache, 0755); err != nil {
		return err
	}
	if err = os.MkdirAll(zvmInstallDirENV, 0755); err != nil {
		return err
	}
	tempDownload, err := os.CreateTemp(z.Directories.cache, fmt.Sprintf("*.%s", archive))
	if err != nil {
		return err
	}
	defer tempDownload.Close()
	defer os.Remove(tempDownload.Name())

	log.Debug("tempDir", "name", tempDownload.Name())
	pbar := progressbar.DefaultBytes(
		int64(resp.ContentLength),
		"Upgrading ZVM...",
	)

	_, err = io.Copy(io.MultiWriter(tempDownload, pbar), resp.Body)
	if err != nil {
		return err
	}

	zvmPath := filepath.Join(zvmInstallDirENV, zvmBinaryName)
	if err := os.Remove(filepath.Join(zvmInstallDirENV, zvmBinaryName)); err != nil {
		if err, ok := err.(*os.PathError); ok {
			if os.IsNotExist(err) {
				log.Debug("Failed to remove file", "path", zvmPath)
			}
		}
	}

	log.Debug("zvmPath", "path", zvmPath)

	newTemp, err := os.MkdirTemp(z.Directories.cache, "zvm-upgrade-*")
	if err != nil {
		log.Debugf("Failed to create temp direcory: %s", newTemp)
		return errors.Join(ErrFailedUpgrade, err)
	}

	defer os.RemoveAll(newTemp)

	if runtime.GOOS == "windows" {
		log.Debug("unzip", "from", tempDownload.Name(), "to", newTemp)
		if err := unzipSource(tempDownload.Name(), newTemp); err != nil {
			log.Error(err)
			return err
		}

		secondaryZVM := fmt.Sprintf("%s.old", zvmPath)
		log.Debug("SecondaryZVM", "path", secondaryZVM)

		newDownload := filepath.Join(newTemp, fmt.Sprintf("zvm-%s-%s", runtime.GOOS, runtime.GOARCH), zvmBinaryName)

		if err := replaceExe(newDownload, zvmPath); err != nil {
			log.Warn("This command might break if ZVM is installed outside of ~/.zvm/self/")
			return fmt.Errorf("upgrade error: %q", err)
		}
		// fmt.Println("Run the following to complete your upgrade on Windows.")
		// fmt.Printf("- Command Prompt:\n\tmove /Y '%s' '%s'\n", secondaryZVM, zvmPath)
		// fmt.Printf("- Powershell:\n\tMove-Item -Path '%s' -Destination '%s' -Force\n", secondaryZVM, zvmPath)

	} else {
		if err := untar(tempDownload.Name(), newTemp); err != nil {
			log.Error(err)
			return err
		}

		if err := os.Rename(filepath.Join(newTemp, zvmBinaryName), zvmPath); err != nil {
			log.Debugf("Failed to rename %s to %s", filepath.Join(newTemp, zvmBinaryName), zvmPath)
			return errors.Join(ErrFailedUpgrade, err)
		}
	}

	if err := os.Chmod(zvmPath, 0775); err != nil {
		log.Debugf("Failed to update permissions for %s", zvmPath)
		return errors.Join(ErrFailedUpgrade, err)
	}

	if _, err := os.Lstat(filepath.Join(z.Directories.bin, zvmBinaryName)); err != nil {
		if err := meta.Symlink(zvmPath, filepath.Join(z.Directories.bin, zvmBinaryName)); err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

// Replaces one file with another on Windows.
func replaceExe(from, to string) error {
	if runtime.GOOS == "windows" {
		if err := os.Rename(to, fmt.Sprintf("%s.old", to)); err != nil {
			ret := true
			if errors.Is(err, os.ErrNotExist) {
				path, execErr := os.Executable()
				if execErr != nil {
					// we have a dumpster fire...bail now
					return errors.Join(err, execErr)
				}
				if path != to {
					log.Infof("Could not rename existing zvm.exe, but it appears you are not upgrading in place. Running zvm %s, upgrading zvm in %s", path, to)
					ret = false
				}
			}
			if ret {
				return err
			}
		}
	} else {
		// This logic is not correct, but this function is only being called
		// when runtime.GOOS == "windows"
		if err := os.Remove(to); err != nil {
			return err
		}
	}

	if err := os.Rename(from, to); err != nil {
		from_io, err := os.Open(from)
		if err != nil {
			return err
		}
		defer from_io.Close()

		to_io, err := os.Create(to)
		if err != nil {
			return err
		}
		defer to_io.Close()

		if _, err := io.Copy(to_io, from_io); err != nil {
			return nil
		}
	}

	return nil
}

// getInstallDir finds the directory this executabile is in.
func (z ZVM) getInstallDir() (string, error) {
	// It is a bit unclear what we should do here depending on the exact environment
	// We have three paths to choose from:
	// 1. Native install pathing
	// 2. ZVM_INSTALL
	// 3. Location of our current running executable
	//
	// In the case that ZVM_INSTALL is set, we should use that and ignore
	// anything else
	//
	// If ZVM_INSTALL is unset and the current executable is not at the path
	// of the native install pathing, then what?
	//
	// Current documentation states that the current executable location wins,
	// but that documentation had no concept of native pathing. We can't use
	// current executable location for the upgrade test either...
	//
	// The most reasonable thing may be to use native pathing, and in the case
	// that there is a discrepancy between where the current executable is
	// and where the upgrade happens, we use native pathing and warn the user
	if zvmInstallDir, ok := os.LookupEnv("ZVM_INSTALL"); ok {
		return zvmInstallDir, nil
	}

	this, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("%q: failed to determine executable path: %w", ErrFailedUpgrade, err)
	}

	this, err = filepath.EvalSymlinks(this)
	if err != nil {
		return "", fmt.Errorf("%q: failed to resolve symlinks: %w", ErrFailedUpgrade, err)
	}

	modifyable, err := canModifyFile(this)
	if err != nil {
		return "", fmt.Errorf("%q, couldn't determine permissions to modify zvm install: %w", ErrFailedUpgrade, err)
	}

	if !modifyable {
		return "", fmt.Errorf("%q, zvm executable cannot be upgraded because is not modifyable", ErrFailedUpgrade)
	}

	finalPath := filepath.Dir(this)
	if finalPath != z.Directories.self {
		log.Warnf("We are upgrading zvm in a different directory (%s) than where zvm is currently running (%s)", z.Directories.self, finalPath)
	}
	return z.Directories.self, nil
}

func resolveSymlink(symlink string) (string, error) {
	target, err := os.Readlink(symlink)
	if err != nil {
		return "", err
	}
	// Ensure the path is absolute
	absolutePath, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	return absolutePath, nil
}

func untar(tarball, target string) error {
	log.Debug("untar", "tarball", tarball, "target", target)
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := target + string(os.PathSeparator) + header.Name
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			writer, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(writer, tarReader); err != nil {
				return err
			}
			writer.Close()
		}
	}
}

func isSymlink(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.Mode()&os.ModeSymlink != 0, nil
}

// CanIUpgrade returns 3 values:
// 1. The tag name for the target version (the version to upgrade *TO*
// 2. A boolean to indicate if upgrade is possible
// 3. Error value
func CanIUpgrade() (string, bool, error) {
	release, err := getLatestGitHubRelease("tristanisham", "zvm")
	if err != nil {
		return "", false, err
	}

	if semver.Compare(meta.VERSION, release.TagName) == -1 {
		return release.TagName, true, nil
	}

	return release.TagName, meta.ForceUpgrade, nil
}

// func getGitHubReleases(owner, repo string) ([]GithubRelease, error) {
// 	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	var releases []GithubRelease
// 	err = json.NewDecoder(resp.Body).Decode(&releases)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return releases, nil
// }

func getLatestGitHubRelease(owner, repo string) (*GithubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var release GithubRelease
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return nil, err
	}

	return &release, nil
}

type GithubRelease struct {
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Body            string    `json:"body"`
	ZipballURL      string    `json:"zipball_url"`
	NodeID          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	URL             string    `json:"url"`
	HTMLURL         string    `json:"html_url"`
	TarballURL      string    `json:"tarball_url"`
	AssetsURL       string    `json:"assets_url"`
	UploadURL       string    `json:"upload_url"`
	Assets          []struct {
		UpdatedAt          time.Time   `json:"updated_at"`
		CreatedAt          time.Time   `json:"created_at"`
		Label              interface{} `json:"label"`
		ContentType        string      `json:"content_type"`
		Name               string      `json:"name"`
		URL                string      `json:"url"`
		State              string      `json:"state"`
		NodeID             string      `json:"node_id"`
		BrowserDownloadURL string      `json:"browser_download_url"`
		Uploader           struct {
			FollowingURL      string `json:"following_url"`
			NodeID            string `json:"node_id"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			Login             string `json:"login"`
			Type              string `json:"type"`
			AvatarURL         string `json:"avatar_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			ID                int    `json:"id"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"uploader"`
		Size          int `json:"size"`
		DownloadCount int `json:"download_count"`
		ID            int `json:"id"`
	} `json:"assets"`
	Author struct {
		FollowingURL      string `json:"following_url"`
		NodeID            string `json:"node_id"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		Login             string `json:"login"`
		Type              string `json:"type"`
		AvatarURL         string `json:"avatar_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		ID                int    `json:"id"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	ID         int  `json:"id"`
	Prerelease bool `json:"prerelease"`
	Draft      bool `json:"draft"`
}
