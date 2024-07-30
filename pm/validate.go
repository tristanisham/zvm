package pm

import (
	"fmt"
	"net/url"
)

// Validate checks the zvm.json in the present directory for invalid schema.
func (p Project) Validate() error {
	if p.Url != "" {
		parsed, err := url.Parse(p.Url)
		if err != nil {
			return fmt.Errorf("invalid field \"url\", %w", err)
		}

		if parsed.Scheme != "git" {
			return fmt.Errorf("%w. Expected 'git'", ErrInvalidScheme)
		}
	}


	return nil
}