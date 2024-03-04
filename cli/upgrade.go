// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"os"

	// "os/user"
	"path/filepath"
	"runtime"

	// "syscall"
	"github.com/tristanisham/zvm/cli/meta"
	"time"

	"archive/tar"

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

	upgradable, tagName, err := CanIUpgrade()
	if err != nil {
		return errors.Join(ErrFailedUpgrade, err)
	}

	if !upgradable {
		fmt.Printf("You are already on the latest release (%s) of ZVM :) \n", clr.Blue(meta.VERSION))
		os.Exit(0)
	} else {
		fmt.Printf("You are on ZVM %s... upgrading to (%s)", meta.VERSION, tagName)
	}

	zvmInstallDirENV, err := z.getInstallDir()
	if err != nil {
		return err
	}

	log.Debug("exe dir", "path", zvmInstallDirENV)
	zvmBinaryName := "zvm"
	archive := "tar"
	if runtime.GOOS == "windows" {
		archive = "zip"
		zvmBinaryName = "zvm.exe"
	}

	download := fmt.Sprintf("zvm-%s-%s.%s", runtime.GOOS, runtime.GOARCH, archive)

	downloadUrl := fmt.Sprintf("https://github.com/tristanisham/zvm/releases/latest/download/%s", download)

	resp, err := http.Get(downloadUrl)
	if err != nil {
		errors.Join(ErrFailedUpgrade, err)
	}
	defer resp.Body.Close()

	tempDownload, err := os.CreateTemp(z.baseDir, fmt.Sprintf("*.%s", archive))
	if err != nil {
		return err
	}
	defer tempDownload.Close()
	// log.Debug("temp name", "path", tempDownload.Name())
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

	newTemp, err := os.MkdirTemp(z.baseDir, "zvm-upgrade-*")
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

		secondaryZVM := fmt.Sprintf("%s2", zvmPath)
		log.Debug("SecondaryZVM", "path", secondaryZVM)

		if err := os.Rename(filepath.Join(newTemp, fmt.Sprintf("zvm-%s-%s", runtime.GOOS, runtime.GOARCH), zvmBinaryName), secondaryZVM); err != nil {
			log.Debugf("Failed to rename %s to %s", filepath.Join(newTemp, fmt.Sprintf("zvm-%s-%s", runtime.GOOS, runtime.GOARCH), zvmBinaryName), secondaryZVM)
			return errors.Join(ErrFailedUpgrade, err)
		}

		fmt.Println("Run the following to complete your upgrade on Windows.")
		fmt.Printf("- Command Prompt:\n\tmove /Y %s %s\n", secondaryZVM, zvmPath)
		fmt.Printf("- Powershell:\n\tMove-Item -Path %s -Destination %s -Force\n", secondaryZVM, zvmPath)

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

	return nil
}

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
		if !itIsASymlink {
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

func CanIUpgrade() (bool, string, error) {
	release, err := getLatestGitHubRelease("tristanisham", "zvm")
	if err != nil {
		return false, "", err
	}

	if semver.Compare(meta.VERSION, release.TagName) == -1 {
		return true, release.TagName, nil
	}

	return false, release.TagName, nil
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
	URL       string `json:"url"`
	AssetsURL string `json:"assets_url"`
	UploadURL string `json:"upload_url"`
	HTMLURL   string `json:"html_url"`
	ID        int    `json:"id"`
	Author    struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	NodeID          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Assets          []struct {
		URL      string      `json:"url"`
		ID       int         `json:"id"`
		NodeID   string      `json:"node_id"`
		Name     string      `json:"name"`
		Label    interface{} `json:"label"`
		Uploader struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			NodeID            string `json:"node_id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"uploader"`
		ContentType        string    `json:"content_type"`
		State              string    `json:"state"`
		Size               int       `json:"size"`
		DownloadCount      int       `json:"download_count"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		BrowserDownloadURL string    `json:"browser_download_url"`
	} `json:"assets"`
	TarballURL string `json:"tarball_url"`
	ZipballURL string `json:"zipball_url"`
	Body       string `json:"body"`
}
