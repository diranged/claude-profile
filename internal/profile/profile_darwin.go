//go:build darwin

package profile

import (
	"os"
	"os/exec"
	"os/user"
	"strings"
)

// keychainHas returns true if the macOS keychain contains a generic password
// entry matching this profile's ServiceKey and the current user. Shells out
// to the "security" CLI, which is darwin-only.
func (p *Profile) keychainHas() bool {
	err := exec.Command("security", "find-generic-password",
		"-s", p.ServiceKey, "-a", currentUser()).Run()
	return err == nil
}

// keychainRead returns the raw credential blob stored in the macOS keychain
// for this profile. Caller is expected to have checked keychainHas() first.
func (p *Profile) keychainRead() (string, error) {
	out, err := exec.Command("security", "find-generic-password",
		"-s", p.ServiceKey, "-a", currentUser(), "-w").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// keychainDelete removes this profile's macOS keychain entry. Errors are
// ignored by the caller because the entry may not exist.
func (p *Profile) keychainDelete() error {
	return exec.Command("security", "delete-generic-password",
		"-s", p.ServiceKey, "-a", currentUser()).Run()
}

// currentUser returns the current OS username for use as the keychain account
// name. Falls back to the USER environment variable if user.Current() fails.
func currentUser() string {
	u, err := user.Current()
	if err != nil {
		return os.Getenv("USER")
	}
	return u.Username
}
