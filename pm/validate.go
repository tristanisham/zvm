package pm

import (
	"fmt"
	"os"
	"path/filepath"
)

func Validate() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	buildFile := filepath.Join(cwd, "build.zig.zon")
	data, err := os.ReadFile(buildFile)
	if err != nil {
		return err
	}

	lex, err := Parse(data)
	if err != nil {
		return err
	}

	fmt.Println(lex)

	return nil
}