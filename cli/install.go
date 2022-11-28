package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func Install(version string) error {
	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = "~"
	}
	zvm := filepath.Join(homedir, ".zvm")
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

	rawVersionStructure := make(map[string]map[string]any)
	if err := json.Unmarshal(versions, &rawVersionStructure); err != nil {
		return err
	}

	tarPath, err := getTarPath(version, &rawVersionStructure)
	if err != nil {
		return err
	}

	tarReq, err := http.Get(*tarPath)
	if err != nil {
		return err
	}
	defer tarReq.Body.Close()
	_ = os.MkdirAll(filepath.Join(zvm, version), 0755)
	tarDownloadPath := filepath.Join(zvm, version, fmt.Sprintf("%s.tar.xz", version))
	out, err := os.Create(tarDownloadPath)
	if err != nil {
		return err
	}
	defer out.Close()
	written, err := io.Copy(out, tarReq.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Wrote %d megabytes to %s\n", written/1000, tarDownloadPath)

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
