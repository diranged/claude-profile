package profile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Files worth copying from the default ~/.claude config into a new profile.
// These are user-level configuration files that customize Claude's behavior.
var bootstrapFiles = []string{
	"CLAUDE.md",
	"settings.json",
	"settings.local.json",
}

// DefaultConfigDir returns the path to the user's default Claude config directory.
func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// BootstrapFiles returns a list of files that exist in the default config dir
// and could be copied into a new profile. Returns nil if the default dir
// doesn't exist or has no copyable files.
func BootstrapFiles() []string {
	defaultDir := DefaultConfigDir()
	var found []string
	for _, name := range bootstrapFiles {
		src := filepath.Join(defaultDir, name)
		if _, err := os.Stat(src); err == nil {
			found = append(found, name)
		}
	}
	return found
}

// CopyBootstrapFiles copies the specified files from the default config
// directory into the profile's config directory.
func (p *Profile) CopyBootstrapFiles(files []string) error {
	defaultDir := DefaultConfigDir()
	for _, name := range files {
		src := filepath.Join(defaultDir, name)
		dst := filepath.Join(p.ConfigDir, name)

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copying %s: %w", name, err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	// Preserve the source file's permissions
	info, err := in.Stat()
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}
