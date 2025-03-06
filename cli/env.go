// Copyright 2025 Tristan Isham. All rights reserved.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

func (z *ZVM) Env() error {
	dirs := z.Directories
	output := map[string]string{
		"self":  dirs.self,
		"cache": dirs.cache,
		"data":  dirs.data,
		"state": dirs.state,
		"bin":   dirs.bin,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to marshal directory paths: %v", err)
	}

	return nil
}
