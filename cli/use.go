package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

func (z *ZVM) Use(ver string) error {
	z.loadVersionCache()
	_, err := z.getVersion(ver)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("It looks like %s isn't installed. Would you like to install it? [y/n]\n", ver)
		if getConfirmation() {
			err = z.Install(ver)
		} else {
			return fmt.Errorf("version %s is not installed", ver)
		}
	}

	if err != nil {
		return err
	}

	return z.setBin(ver)
}

func (z *ZVM) setBin(ver string) error {
	version_path := filepath.Join(z.zvmBaseDir, ver)
	if err := os.Remove(filepath.Join(z.zvmBaseDir, "bin")); err != nil {
		log.Warn(err)
	}

	if err := os.Symlink(version_path, filepath.Join(z.zvmBaseDir, "bin")); err != nil {
		return err
	}

	return nil
}

func getConfirmation() bool {
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	answer := strings.TrimSpace(strings.ToLower(text))
	return answer == "y" || answer == "ye" || answer == "yes"

}
