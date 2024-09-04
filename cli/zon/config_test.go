package zon

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestConfigParse(t *testing.T) {
	configStr := `.{
		.name = "demo",
		.version = "0.0.0",
		.minimum_zig_version = "0.11.0",
		.dependencies = .{
			.example = .{
				.url = "https://example.com/foo.tar.gz",
				.hash = "...",
				.lazy = false,
			},
		},
		.paths = .{
			"",
			"build.zig",
			"src",
		},
	}`

	configData := &Config{
		Name:              "demo",
		Version:           "0.0.0",
		MinimumZigVersion: "0.11.0",
		Dependencies:      map[string]Dependency{
			"example": {
				URL:  "https://example.com/foo.tar.gz",
				Hash: "...",
				Lazy: false,
			},
		},
		Paths:             []string{
			"",
			"build.zig",
			"src",
		},
	}

	configParsed, err := ParseConfig(strings.NewReader(configStr))
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("Name: %s\n", configParsed.Name)
	fmt.Printf("Version: %s\n", configParsed.Version)
	fmt.Printf("Minimum Zig Version: %s\n", configParsed.MinimumZigVersion)
	fmt.Println("Dependencies:")
	for name, dep := range configParsed.Dependencies {
		fmt.Printf("  %s:\n", name)
		fmt.Printf("    URL: %s\n", dep.URL)
		fmt.Printf("    Hash: %s\n", dep.Hash)
		fmt.Printf("    Path: %s\n", dep.Path)
		fmt.Printf("    Lazy: %v\n", dep.Lazy)
	}
	fmt.Println("Paths:")
	for _, path := range configParsed.Paths {
		fmt.Printf("  %s\n", path)
	}

	if !reflect.DeepEqual(configData, configParsed) {
		t.Errorf("source and parsed structs are not equal")
	}
}