//go:build !darwin

package profile

import "errors"

// keychainHas always returns false on non-darwin builds. Claude Code stores
// credentials in <CLAUDE_CONFIG_DIR>/.credentials.json on these platforms, so
// AuthStatus() and OAuthDetails() fall through to the file path.
func (p *Profile) keychainHas() bool {
	return false
}

// keychainRead is unreachable on non-darwin builds because keychainHas()
// always returns false. Returns an error for safety in case a future caller
// invokes it without checking.
func (p *Profile) keychainRead() (string, error) {
	return "", errors.New("keychain not available on this platform")
}

// keychainDelete is a no-op on non-darwin builds.
func (p *Profile) keychainDelete() error {
	return nil
}
