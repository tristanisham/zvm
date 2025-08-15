// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"

	"github.com/tristanisham/clr"
)

type Settings struct {
	path               string
	MirrorListUrl      string `json:"mirrorListUrl,omitempty"` // Zig's community mirror list URL
	MinisignPubKey     string `json:"minisignPubKey,omitempty"`
	VersionMapUrl      string `json:"versionMapUrl,omitempty"`    // Zig's version map URL
	ZlsVMU             string `json:"zlsVersionMapUrl,omitempty"` // ZLS's version map URL
	UseColor           bool   `json:"useColor"`
	AlwaysForceInstall bool   `json:"alwaysForceInstall"`
}

var DefaultSettings = Settings{
	MirrorListUrl: "https://ziglang.org/download/community-mirrors.txt",
	// From https://ziglang.org/download/
	MinisignPubKey:     "RWSGOq2NVecA2UPNdBUZykf1CCb147pkmdtYxgb3Ti+JO/wCYvhbAb/U",
	VersionMapUrl:      "https://ziglang.org/download/index.json",
	ZlsVMU:             "https://releases.zigtools.org/",
	UseColor:           true,
	AlwaysForceInstall: false,
}

func (s *Settings) UseMirrorList() bool {
	return s.MirrorListUrl != "disabled"
}

func (s *Settings) ToggleColor() {
	s.UseColor = !s.UseColor
	if err := s.save(); err != nil {
		log.Fatal(err)
	}

	if s.UseColor {
		fmt.Printf("Terminal color output: %s\n", clr.Green("ON"))
		return
	}

	fmt.Println("Terminal color output: OFF")
}

func (s *Settings) ResetMirrorList() error {
	s.MirrorListUrl = DefaultSettings.MirrorListUrl
	if err := s.save(); err != nil {
		return err
	}

	return nil
}

func (s *Settings) ResetVersionMap() error {
	s.VersionMapUrl = DefaultSettings.VersionMapUrl
	if err := s.save(); err != nil {
		return err
	}

	return nil
}

func (s *Settings) ResetZlsVMU() error {
	s.ZlsVMU = DefaultSettings.ZlsVMU
	if err := s.save(); err != nil {
		return err
	}

	return nil
}

func (s *Settings) NoColor() {
	s.UseColor = false
	if err := s.save(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Terminal color output: OFF")
}

func (s *Settings) YesColor() {
	s.UseColor = true
	if err := s.save(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Terminal color output: %s\n", clr.Green("ON"))
}

func (s *Settings) SetColor(answer bool) {
	s.UseColor = answer
	if err := s.save(); err != nil {
		log.Fatal(err)
	}
}

func (s *Settings) SetMirrorListUrl(mirrorListUrl string) error {
	if mirrorListUrl != "disabled" {
		if err := isValidWebURL(mirrorListUrl); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidVersionMap, err)
		}
	}

	s.MirrorListUrl = mirrorListUrl
	if err := s.save(); err != nil {
		return err
	}

	log.Debug("set mirror list url", "url", s.MirrorListUrl)

	return nil
}

func (s *Settings) SetVersionMapUrl(versionMapUrl string) error {
	if err := isValidWebURL(versionMapUrl); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidVersionMap, err)
	}

	s.VersionMapUrl = versionMapUrl
	if err := s.save(); err != nil {
		return err
	}

	log.Debug("set version map url", "url", s.VersionMapUrl)

	return nil
}

func (s *Settings) SetZlsVMU(versionMapUrl string) error {
	if err := isValidWebURL(versionMapUrl); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidVersionMap, err)
	}

	s.ZlsVMU = versionMapUrl
	if err := s.save(); err != nil {
		return err
	}

	log.Debug("set zls version map url", "url", s.ZlsVMU)

	return nil
}

func (s *Settings) ResetEmpty() error {
	if s.MirrorListUrl == "" {
		s.MirrorListUrl = DefaultSettings.MirrorListUrl
	}

	if s.MinisignPubKey == "" {
		s.MinisignPubKey = DefaultSettings.MinisignPubKey
	}

	if s.VersionMapUrl == "" {
		s.VersionMapUrl = DefaultSettings.VersionMapUrl
	}

	if s.ZlsVMU == "" {
		s.ZlsVMU = DefaultSettings.ZlsVMU
	}

	return s.save()
}

// isValidWebURL checks if the given URL string is a valid web URL.
func isValidWebURL(urlString string) error {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return err // URL parsing error
	}

	// Check for valid HTTP/HTTPS scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}

	// Check for non-empty host (domain)
	if parsedURL.Host == "" {
		return fmt.Errorf("URL host (domain) is empty")
	}

	// Optionally, you can add more checks (like path, query params, etc.) here if needed

	return nil // URL is valid
}

func (s Settings) save() error {
	outSettings, err := json.MarshalIndent(&s, "", "    ")
	if err != nil {
		return fmt.Errorf("unable to generate settings.json file %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("unable to create settings directory: %w", err)
	}

	if err := os.WriteFile(s.path, outSettings, 0755); err != nil {
		return fmt.Errorf("unable to create settings.json file %w", err)
	}

	return nil
}
