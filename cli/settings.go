// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/charmbracelet/log"

	"github.com/tristanisham/clr"
)

type Settings struct {
	basePath      string
	UseColor      bool   `json:"useColor"`
	VersionMapUrl string `json:"versionMapUrl,omitempty"`
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

func (s *Settings) ResetVersionMap() error {
	s.VersionMapUrl = "https://ziglang.org/download/index.json"
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
	out_settings, err := json.MarshalIndent(&s, "", "    ")
	if err != nil {
		return fmt.Errorf("unable to generate settings.json file %v", err)
	}

	if err := os.WriteFile(s.basePath, out_settings, 0755); err != nil {
		return fmt.Errorf("unable to create settings.json file %v", err)
	}

	return nil
}
