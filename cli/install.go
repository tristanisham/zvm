package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/tristanisham/clr"
)

func (z *ZVM) Install(version string) error {
	zvm := z.zvmBaseDir
	os.Mkdir(zvm, 0755)

	req, err := http.NewRequest("GET", "https://ziglang.org/download/index.json", nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "zvm (Zig Version Manager) 0.0.2")
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	versions, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(zvm, "versions.json"), versions, 0755); err != nil {
		return err
	}

	rawVersionStructure := make(zigVersionMap)
	if err := json.Unmarshal(versions, &rawVersionStructure); err != nil {
		return err
	}
	z.zigVersions = rawVersionStructure

	tarPath, err := getTarPath(version, &rawVersionStructure)
	if err != nil {
		return err
	}

	tarReq, err := http.Get(*tarPath)
	if err != nil {
		return err
	}
	defer tarReq.Body.Close()
	// _ = os.MkdirAll(filepath.Join(zvm, version), 0755)
	// tarDownloadPath := filepath.Join(zvm, version, fmt.Sprintf("%s.tar.xz", version))

	out, err := os.CreateTemp(zvm, "*.tar.xz")
	if err != nil {
		return err
	}
	defer out.Close()
	defer os.RemoveAll(out.Name())

	pbar := progressbar.DefaultBytes(
		tarReq.ContentLength,
		fmt.Sprintf("Downloading %s:", clr.Green(version)),
	)

	_, err = io.Copy(io.MultiWriter(out, pbar), tarReq.Body)
	if err != nil {
		return err
	}

	// The base directory where all Zig files for the appropriate version are installed
	// installedVersionPath := filepath.Join(zvm, version)
	fmt.Println("Extracting bundle...")

	if err := extractTarXZ(out.Name(), zvm); err != nil {
		log.Fatal(clr.Red(err))
	}
	tarName := strings.TrimPrefix(*tarPath, "https://ziglang.org/builds/")
	tarName = strings.TrimSuffix(tarName, ".tar.xz")
	if err := os.Rename(filepath.Join(zvm, tarName), filepath.Join(zvm, version)); err != nil {
		if _, err := os.Stat(filepath.Join(zvm, version)); os.IsExist(err) {
			if err := os.Remove(filepath.Join(zvm, version)); err != nil {
				log.Fatalln(clr.Red(err))
			} else {
				if err := os.Rename(filepath.Join(zvm, tarName), filepath.Join(zvm, version)); err != nil {
					log.Fatalln(clr.Yellow(err))
				}
			}
		}
	}

	// This removes the extra download
	if err := os.RemoveAll(filepath.Join(zvm, tarName)); err != nil {
		log.Println(clr.Red(err))
	}

	if err := os.Remove(filepath.Join(zvm, "bin")); err != nil {
		log.Println(clr.Yellow(err))
	}

	if err := os.Symlink(filepath.Join(zvm, version), filepath.Join(zvm, "bin")); err != nil {
		log.Fatal(clr.Red(err))
	}

	return nil
}

func getTarPath(version string, data *map[string]map[string]any) (*string, error) {
	if info, ok := (*data)[version]; ok {
		arch, ops := zigStyleSysInfo()
		if systemInfo, ok := info[fmt.Sprintf("%s-%s", arch, ops)]; ok {
			if base, ok := systemInfo.(map[string]any); ok {
				if tar, ok := base["tarball"].(string); ok {
					return &tar, nil
				}
			} else {
				return nil, fmt.Errorf("unable to find necessary download path")
			}
		} else {
			return nil, fmt.Errorf("invalid/unsupported system: ARCH: %s OS: %s", arch, ops)
		}
	}
	return nil, fmt.Errorf("invalid Zig version: %s", version)
}

func zigStyleSysInfo() (string, string) {
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "x86_64"
	default:
		arch = runtime.GOARCH
	}

	return arch, runtime.GOOS
}

func extractTarXZ(bundle, out string) error {
	tar := exec.Command("tar", "-xf", bundle, "-C", out)
	tar.Stdout = os.Stdout
	tar.Stderr = os.Stderr
	if err := tar.Run(); err != nil {
		return err
	}

	return nil
}
