package profile

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	profileNameRE = regexp.MustCompile(`^[a-z0-9_-]+$`)
	// reservedProfileNames is the closed set of tokens the CLI keeps for
	// future use. Keeping `default`, `current`, etc. unowned means we can
	// later use them to mean "the active profile" without breaking
	// existing config.
	reservedProfileNames = map[string]struct{}{
		"default": {}, "current": {}, "me": {}, "self": {}, "all": {}, "none": {},
	}
)

func validateProfileName(name string) error {
	if name == "" {
		return errors.New("profile name is required")
	}
	if len(name) > 32 {
		return fmt.Errorf("profile name %q is longer than 32 chars", name)
	}
	if !profileNameRE.MatchString(name) {
		return fmt.Errorf("profile name %q must match [a-z0-9_-]+", name)
	}
	if _, reserved := reservedProfileNames[name]; reserved {
		return fmt.Errorf("profile name %q is reserved", name)
	}
	return nil
}
