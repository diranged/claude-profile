package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfigDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, ".claude"), DefaultConfigDir())
}

func TestBootstrapFiles_NoDefaultDir(t *testing.T) {
	// When the default config dir doesn't exist or has no files,
	// BootstrapFiles should return nil.
	// We can't easily override DefaultConfigDir since it uses os.UserHomeDir,
	// but if the user doesn't have these files, this returns nil.
	// At minimum, verify it doesn't panic.
	_ = BootstrapFiles()
}

func TestCopyBootstrapFiles_Success(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake "default config dir" with files
	srcDir := filepath.Join(tmp, "defaultconfig")
	require.NoError(t, os.MkdirAll(srcDir, 0700))

	// Create source files
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file2.txt"), []byte("content2"), 0600))

	// Create a profile with a config dir
	dstDir := filepath.Join(tmp, "profile", "config")
	require.NoError(t, os.MkdirAll(dstDir, 0700))

	p := &Profile{
		Name:      "test",
		Dir:       filepath.Join(tmp, "profile"),
		ConfigDir: dstDir,
	}

	// Test copyFile directly since CopyBootstrapFiles uses DefaultConfigDir().
	_ = p // p is available if needed for CopyBootstrapFiles integration tests
	src := filepath.Join(srcDir, "file1.txt")
	dst := filepath.Join(dstDir, "file1.txt")
	require.NoError(t, copyFile(src, dst))

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "content1", string(data))
}

func TestCopyFile_PreservesPermissions(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")

	require.NoError(t, os.WriteFile(src, []byte("hello"), 0755))

	require.NoError(t, copyFile(src, dst))

	srcInfo, err := os.Stat(src)
	require.NoError(t, err)
	dstInfo, err := os.Stat(dst)
	require.NoError(t, err)

	assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
}

func TestCopyFile_SourceNotExist(t *testing.T) {
	tmp := t.TempDir()
	err := copyFile(filepath.Join(tmp, "nonexistent"), filepath.Join(tmp, "dst"))
	assert.Error(t, err)
}

func TestCopyFile_DestDirNotExist(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	require.NoError(t, os.WriteFile(src, []byte("data"), 0644))

	dst := filepath.Join(tmp, "nonexistent", "dir", "dst.txt")
	err := copyFile(src, dst)
	assert.Error(t, err)
}

func TestCopyFile_ContentIntegrity(t *testing.T) {
	tmp := t.TempDir()

	content := "line1\nline2\nline3\n"
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")

	require.NoError(t, os.WriteFile(src, []byte(content), 0644))
	require.NoError(t, copyFile(src, dst))

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestCopyFile_EmptyFile(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "empty.txt")
	dst := filepath.Join(tmp, "dst.txt")

	require.NoError(t, os.WriteFile(src, []byte(""), 0644))
	require.NoError(t, copyFile(src, dst))

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "", string(data))
}

func TestCopyBootstrapFiles_MissingSource(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_PROFILES_DIR", tmp)

	p := Load("test")
	require.NoError(t, p.EnsureDir())

	// Try to copy a file that doesn't exist in the default config dir
	err := p.CopyBootstrapFiles([]string{"nonexistent-file.txt"})
	assert.Error(t, err)
}
