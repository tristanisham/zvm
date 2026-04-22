// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// You can disable auto-updates by setting `-tags noAutoUpgrades` when building.
//go:build !noAutoUpgrades

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
	"strings"
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
		return fmt.Errorf("%w: %w", ErrFailedUpgrade, err)
	}

	if !upgradable {
		fmt.Printf("You are already on the latest release (%s) of ZVM :) \n", clr.Blue(meta.VERSION))
		return nil
	}
	fmt.Printf("You are on ZVM %s... upgrading to (%s)", meta.VERSION, tagName)

	zvmInstallDirENV, err := z.getInstallDir()
	if err != nil {
		return err
	}

	log.Debug("exe dir", "path", zvmInstallDirENV)
	zvmBinaryName := "zvm"
	archive := "tar"
	if runtime.GOOS == "windows" {
		zvmBinaryName = "zvm.exe"
		archive = "zip"
	}

	downloadUrl := fmt.Sprintf(
		"https://github.com/tristanisham/zvm/releases/latest/download/zvm-%s-%s.%s",
		runtime.GOOS, runtime.GOARCH, archive,
	)

	resp, err := http.Get(downloadUrl)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedUpgrade, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: unexpected status code %d", ErrFailedUpgrade, resp.StatusCode)
	}

	tempDownload, err := os.CreateTemp(z.baseDir, "*."+archive)
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
	log.Debug("zvmPath", "path", zvmPath)

	newTemp, err := os.MkdirTemp(z.baseDir, "zvm-upgrade-*")
	if err != nil {
		log.Debugf("Failed to create temp directory: %s", newTemp)
		return fmt.Errorf("%w: %w", ErrFailedUpgrade, err)
	}
	defer os.RemoveAll(newTemp)

	switch archive {
	case "zip":
		log.Debug("unzip", "from", tempDownload.Name(), "to", newTemp)
		if err := unzipSource(tempDownload.Name(), newTemp); err != nil {
			return fmt.Errorf("%w: %w", ErrFailedUpgrade, err)
		}
	case "tar":
		log.Debug("untar", "from", tempDownload.Name(), "to", newTemp)
		if err := untar(tempDownload.Name(), newTemp); err != nil {
			return fmt.Errorf("%w: %w", ErrFailedUpgrade, err)
		}
	}

	src := filepath.Join(newTemp, zvmBinaryName)

	if err := replaceExe(src, zvmPath); err != nil {
		log.Warn("This command might break if ZVM is installed outside of ~/.zvm/self/")
		return fmt.Errorf("%w: %w", ErrFailedUpgrade, err)
	}

	// Clean up the .old backup from Windows upgrades (best-effort)
	if runtime.GOOS == "windows" {
		os.Remove(fmt.Sprintf("%s.old", zvmPath))
	}

	if err := os.Chmod(zvmPath, 0775); err != nil {
		log.Debugf("Failed to update permissions for %s", zvmPath)
		return fmt.Errorf("%w: %w", ErrFailedUpgrade, err)
	}

	return nil
}

// replaceExe replaces the file at `to` with the file at `from`.
// The existing file is renamed to `.old` as a backup so it can be
// restored if the replacement fails.
func replaceExe(from, to string) error {
	oldPath := fmt.Sprintf("%s.old", to)

	if err := os.Rename(to, oldPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if err := os.Rename(from, to); err != nil {
		// Cross-device fallback: copy the file contents
		if copyErr := copyFile(from, to); copyErr != nil {
			// Rollback: restore the old binary
			if rbErr := os.Rename(oldPath, to); rbErr != nil {
				log.Error("Failed to rollback after upgrade failure", "err", rbErr)
			}
			return copyErr
		}
	}

	return nil
}

// copyFile copies the contents of src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// getInstallDir finds the directory this executabile is in.
func (z ZVM) getInstallDir() (string, error) {
	zvmInstallDirENV, ok := os.LookupEnv("ZVM_INSTALL")
	if !ok {
		this, err := os.Executable()
		if err != nil {
			return filepath.Join(z.baseDir, "self"), nil
		}

		itIsASymlink, err := isSymlink(this)
		if err != nil {
			return filepath.Join(z.baseDir, "self"), nil
		}

		var finalPath string
		if itIsASymlink {
			finalPath, err = resolveSymlink(this)
			if err != nil {
				return filepath.Join(z.baseDir, "self"), nil
			}
		} else {
			finalPath = this
		}

		modifyable, err := canModifyFile(finalPath)
		if err != nil {
			return "", fmt.Errorf("%q, couldn't determine permissions to modify zvm install", ErrFailedUpgrade)
		}

		if modifyable {
			return filepath.Dir(this), nil
		}

		return "", fmt.Errorf("%q, didn't have permissions to modify zvm install", ErrFailedUpgrade)
	}

	return zvmInstallDirENV, nil
}

// resolveSymlink follows a symbolic link and returns the absolute path to the target.
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

// untar extracts a tarball to the specified target directory.
func untar(tarball, target string) error {
	log.Debug("untar", "tarball", tarball, "target", target)
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}

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

		fpath := filepath.Join(absTarget, header.Name)

		if !strings.HasPrefix(fpath, absTarget+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(fpath); err != nil {
				if err := os.MkdirAll(fpath, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return err
			}

			writer, err := os.Create(fpath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(writer, tarReader); err != nil {
				writer.Close()
				return err
			}
			writer.Close()
		}
	}
}

// isSymlink checks if the given path is a symbolic link.
func isSymlink(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.Mode()&os.ModeSymlink != 0, nil
}

// CanIUpgrade checks if a newer version of ZVM is available on GitHub.
// It returns the latest tag name, a boolean indicating if an upgrade is available, and any error.
func CanIUpgrade() (string, bool, error) {
	release, err := getLatestGitHubRelease("tristanisham", "zvm")
	if err != nil {
		return "", false, err
	}

	if semver.Compare(meta.VERSION, release.TagName) == -1 {
		return release.TagName, true, nil
	}

	return release.TagName, false, nil
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

// getLatestGitHubRelease fetches the latest release information for the specified repository from GitHub API.
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

// GithubRelease represents the JSON structure of a GitHub release object.
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
		UpdatedAt          time.Time `json:"updated_at"`
		CreatedAt          time.Time `json:"created_at"`
		Label              any       `json:"label"`
		ContentType        string    `json:"content_type"`
		Name               string    `json:"name"`
		URL                string    `json:"url"`
		State              string    `json:"state"`
		NodeID             string    `json:"node_id"`
		BrowserDownloadURL string    `json:"browser_download_url"`
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
