package cli

import (
	"os"
	"path/filepath"
	"testing"
)

type pathTest struct {
	name           string
	home           string
	pathFunc       func(string) Directories
	expectedData   string
	expectedConfig string
	expectedState  string
	expectedBin    string
	expectedCache  string
	setupEnv       func()
	cleanupEnv     func()
}

func TestAllDefaultPaths(t *testing.T) {
	// Save original env vars and restore after test
	originalEnv := map[string]string{
		"ZVM_PATH":        os.Getenv("ZVM_PATH"),
		"XDG_DATA_HOME":   os.Getenv("XDG_DATA_HOME"),
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"XDG_STATE_HOME":  os.Getenv("XDG_STATE_HOME"),
		"XDG_CACHE_HOME":  os.Getenv("XDG_CACHE_HOME"),
		"XDG_BIN_HOME":    os.Getenv("XDG_BIN_HOME"),
		"APPDATA":         os.Getenv("APPDATA"),
		"LOCALAPPDATA":    os.Getenv("LOCALAPPDATA"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	tests := []pathTest{
		{
			name:     "darwin",
			home:     filepath.Join("/Users", "testuser"),
			pathFunc: darwinDirectories,
			expectedData:   filepath.Join("/Users", "testuser", "Library", "Application Support", "zvm"),
			expectedConfig: filepath.Join("/Users", "testuser", "Library", "Preferences", "zvm"),
			expectedState:  filepath.Join("/Users", "testuser", "Library", "Application Support", "zvm"),
			expectedBin:    filepath.Join("/Users", "testuser", ".local", "bin"),
			expectedCache:  filepath.Join("/Users", "testuser", "Library", "Caches", "zvm"),
		},
		{
			name:     "windows",
			home:     filepath.Join("C:", "Users", "TestUser"),
			pathFunc: windowsDirectories,
			expectedData:   filepath.Join("C:", "Users", "TestUser", "AppData", "Local", "zvm"),
			expectedConfig: filepath.Join("C:", "Users", "TestUser", "AppData", "Roaming", "zvm"),
			expectedState:  filepath.Join("C:", "Users", "TestUser", "AppData", "Local", "zvm"),
			expectedBin:    filepath.Join("C:", "Users", "TestUser", "AppData", "Local", "bin"),
			expectedCache:  filepath.Join("C:", "Users", "TestUser", "AppData", "Local", "zvm", "cache"),
			setupEnv: func() {
				os.Setenv("APPDATA", filepath.Join("C:", "Users", "TestUser", "AppData", "Roaming"))
				os.Setenv("LOCALAPPDATA", filepath.Join("C:", "Users", "TestUser", "AppData", "Local"))
			},
		},
		{
			name:     "plan9",
			home:     filepath.Join("/usr", "testuser"),
			pathFunc: plan9Directories,
			expectedData:   filepath.Join("/usr", "testuser", ".zvm"),
			expectedConfig: filepath.Join("/usr", "testuser", ".zvm"),
			expectedState:  filepath.Join("/usr", "testuser", ".zvm"),
			expectedBin:    filepath.Join("/usr", "testuser", "bin"),
			expectedCache:  filepath.Join("/usr", "testuser", ".zvm", "cache"),
		},
		{
			name:     "unix",
			home:     filepath.Join("/home", "testuser"),
			pathFunc: unixDirectories,
			expectedData:   filepath.Join("/home", "testuser", ".local", "share", "zvm"),
			expectedConfig: filepath.Join("/home", "testuser", ".config", "zvm"),
			expectedState:  filepath.Join("/home", "testuser", ".local", "state", "zvm"),
			expectedBin:    filepath.Join("/home", "testuser", ".local", "bin"),
			expectedCache:  filepath.Join("/home", "testuser", ".cache", "zvm"),
		},
		{
			name:     "unix with XDG vars",
			home:     filepath.Join("/home", "testuser"),
			pathFunc: unixDirectories,
			expectedData:   filepath.Join("/custom", "data", "zvm"),
			expectedConfig: filepath.Join("/custom", "config", "zvm"),
			expectedState:  filepath.Join("/custom", "state", "zvm"),
			expectedBin:    filepath.Join("/custom", "bin"),
			expectedCache:  filepath.Join("/custom", "cache", "zvm"),
			setupEnv: func() {
				os.Setenv("XDG_DATA_HOME", filepath.Join("/custom", "data"))
				os.Setenv("XDG_CONFIG_HOME", filepath.Join("/custom", "config"))
				os.Setenv("XDG_STATE_HOME", filepath.Join("/custom", "state"))
				os.Setenv("XDG_BIN_HOME", filepath.Join("/custom", "bin"))
				os.Setenv("XDG_CACHE_HOME", filepath.Join("/custom", "cache"))
			},
		},
		{
			name:     "with ZVM_PATH",
			home:     filepath.Join("/home", "testuser"),
			pathFunc: unixDirectories,
			expectedData:   filepath.Join("/opt", "zvm"),
			expectedConfig: filepath.Join("/opt", "zvm"),
			expectedState:  filepath.Join("/opt", "zvm"),
			expectedBin:    filepath.Join("/home", "testuser", ".local", "bin"),
			expectedCache:  filepath.Join("/opt", "zvm"),
			setupEnv: func() {
				os.Setenv("ZVM_PATH", filepath.Join("/opt", "zvm"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars
			for k := range originalEnv {
				os.Unsetenv(k)
			}

			if tt.setupEnv != nil {
				tt.setupEnv()
			}

			directories := tt.pathFunc(tt.home)

			if directories.data != tt.expectedData {
				t.Errorf("data path = %v, want %v", directories.data, tt.expectedData)
			}
			if directories.config != tt.expectedConfig {
				t.Errorf("config path = %v, want %v", directories.config, tt.expectedConfig)
			}
			if directories.state != tt.expectedState {
				t.Errorf("state path = %v, want %v", directories.state, tt.expectedState)
			}
			if directories.bin != tt.expectedBin {
				t.Errorf("bin path = %v, want %v", directories.bin, tt.expectedBin)
			}
			if directories.cache != tt.expectedCache {
				t.Errorf("cache path = %v, want %v", directories.cache, tt.expectedCache)
			}

			if tt.cleanupEnv != nil {
				tt.cleanupEnv()
			}
		})
	}
}
