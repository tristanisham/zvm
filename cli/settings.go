package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/tristanisham/clr"
)

type Settings struct {
	basePath            string
	UseColor            bool `json:"useColor"`
	StartupCheckUpgrade bool `json:"startupCheckUpgrade"`
	VersionRepo	string `json:"versionRepo"`
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
