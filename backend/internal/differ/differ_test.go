package differ

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTarball creates a .tar.gz file with the given files (path -> content).
func createTarball(t *testing.T, files map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "package.tar.gz")

	f, err := os.Create(tarPath)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	return tarPath
}

func TestExtractTarball_Valid(t *testing.T) {
	tarPath := createTarball(t, map[string]string{
		"pkg-1.0/setup.py":    "from setuptools import setup\nsetup(name='pkg')\n",
		"pkg-1.0/README.md":   "# My Package\n",
		"pkg-1.0/src/main.py": "print('hello')\n",
	})

	dir, err := extractTarball(tarPath)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	content, err := os.ReadFile(filepath.Join(dir, "pkg-1.0", "setup.py"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "setuptools")

	content, err = os.ReadFile(filepath.Join(dir, "pkg-1.0", "src", "main.py"))
	require.NoError(t, err)
	assert.Equal(t, "print('hello')\n", string(content))
}

func TestExtractTarball_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "malicious.tar.gz")

	f, err := os.Create(tarPath)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	content := "safe content"
	tw.WriteHeader(&tar.Header{Name: "pkg/safe.txt", Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg})
	tw.Write([]byte(content))

	malicious := "evil content"
	tw.WriteHeader(&tar.Header{Name: "../../../etc/passwd", Mode: 0o644, Size: int64(len(malicious)), Typeflag: tar.TypeReg})
	tw.Write([]byte(malicious))

	tw.Close()
	gw.Close()
	f.Close()

	dir, err := extractTarball(tarPath)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	_, err = os.ReadFile(filepath.Join(dir, "pkg", "safe.txt"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "..", "..", "..", "etc", "passwd"))
	assert.True(t, os.IsNotExist(err), "path traversal file should not be extracted")
}

func TestExtractTarball_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	badPath := filepath.Join(tmpDir, "not-a-tarball.tar.gz")
	os.WriteFile(badPath, []byte("not gzip data"), 0o644)

	_, err := extractTarball(badPath)
	assert.Error(t, err)
}

func TestExtractTarball_NonexistentFile(t *testing.T) {
	_, err := extractTarball("/nonexistent/file.tar.gz")
	assert.Error(t, err)
}

func TestExtractTarball_EmptyArchive(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "empty.tar.gz")

	f, err := os.Create(tarPath)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	tw.Close()
	gw.Close()
	f.Close()

	dir, err := extractTarball(tarPath)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestExtractTarball_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "withdir.tar.gz")

	f, err := os.Create(tarPath)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	tw.WriteHeader(&tar.Header{Name: "pkg/", Typeflag: tar.TypeDir, Mode: 0o755})

	content := "file in dir"
	tw.WriteHeader(&tar.Header{Name: "pkg/file.txt", Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg})
	tw.Write([]byte(content))

	tw.Close()
	gw.Close()
	f.Close()

	dir, err := extractTarball(tarPath)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	data, err := os.ReadFile(filepath.Join(dir, "pkg", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "file in dir", string(data))
}

func TestWalkFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "pkg-1.0", "sub"), 0o755))
	os.WriteFile(filepath.Join(dir, "pkg-1.0", "file1.py"), []byte("content1"), 0o644)
	os.WriteFile(filepath.Join(dir, "pkg-1.0", "sub", "file2.py"), []byte("content2"), 0o644)

	files, err := walkFiles(dir)
	require.NoError(t, err)

	assert.Equal(t, "content1", files["file1.py"])
	assert.Equal(t, "content2", files[filepath.Join("sub", "file2.py")])
	assert.Len(t, files, 2)
}

func TestWalkFiles_SkipsBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "pkg"), 0o755))

	os.WriteFile(filepath.Join(dir, "pkg", "small.txt"), []byte("small"), 0o644)

	largeContent := make([]byte, 1024*1024+1)
	os.WriteFile(filepath.Join(dir, "pkg", "large.bin"), largeContent, 0o644)

	files, err := walkFiles(dir)
	require.NoError(t, err)

	assert.Contains(t, files, "small.txt")
	assert.NotContains(t, files, "large.bin")
}

func TestWalkFiles_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	files, err := walkFiles(dir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestGenerateDiff_NewFile(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "pkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "pkg"), 0o755))

	os.WriteFile(filepath.Join(oldDir, "pkg", "existing.py"), []byte("print('old')"), 0o644)
	os.WriteFile(filepath.Join(newDir, "pkg", "existing.py"), []byte("print('old')"), 0o644)
	os.WriteFile(filepath.Join(newDir, "pkg", "new.py"), []byte("print('new')"), 0o644)

	diff, stats, err := generateDiff(oldDir, newDir)
	require.NoError(t, err)

	assert.Equal(t, 1, stats.filesChanged)
	assert.Greater(t, stats.linesAdded, 0)
	assert.Equal(t, 0, stats.linesRemoved)
	assert.Contains(t, diff, "+++ b/new.py")
	assert.Contains(t, diff, "+print('new')")
}

func TestGenerateDiff_DeletedFile(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "pkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "pkg"), 0o755))

	os.WriteFile(filepath.Join(oldDir, "pkg", "removed.py"), []byte("old code"), 0o644)

	diff, stats, err := generateDiff(oldDir, newDir)
	require.NoError(t, err)

	assert.Equal(t, 1, stats.filesChanged)
	assert.Equal(t, 0, stats.linesAdded)
	assert.Greater(t, stats.linesRemoved, 0)
	assert.Contains(t, diff, "--- a/removed.py")
	assert.Contains(t, diff, "+++ /dev/null")
}

func TestGenerateDiff_ModifiedFile(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "pkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "pkg"), 0o755))

	os.WriteFile(filepath.Join(oldDir, "pkg", "main.py"), []byte("v1"), 0o644)
	os.WriteFile(filepath.Join(newDir, "pkg", "main.py"), []byte("v2"), 0o644)

	diff, stats, err := generateDiff(oldDir, newDir)
	require.NoError(t, err)

	assert.Equal(t, 1, stats.filesChanged)
	assert.Greater(t, stats.linesAdded, 0)
	assert.Greater(t, stats.linesRemoved, 0)
	assert.Contains(t, diff, "--- a/main.py")
	assert.Contains(t, diff, "+++ b/main.py")
}

func TestGenerateDiff_NoChanges(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "pkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "pkg"), 0o755))

	os.WriteFile(filepath.Join(oldDir, "pkg", "same.py"), []byte("identical"), 0o644)
	os.WriteFile(filepath.Join(newDir, "pkg", "same.py"), []byte("identical"), 0o644)

	diff, stats, err := generateDiff(oldDir, newDir)
	require.NoError(t, err)

	assert.Equal(t, 0, stats.filesChanged)
	assert.Equal(t, 0, stats.linesAdded)
	assert.Equal(t, 0, stats.linesRemoved)
	assert.Empty(t, diff)
}

func TestGenerateDiff_MultipleChanges(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "pkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "pkg"), 0o755))

	os.WriteFile(filepath.Join(oldDir, "pkg", "a.py"), []byte("old"), 0o644)
	os.WriteFile(filepath.Join(newDir, "pkg", "a.py"), []byte("new"), 0o644)
	os.WriteFile(filepath.Join(newDir, "pkg", "b.py"), []byte("added"), 0o644)
	os.WriteFile(filepath.Join(oldDir, "pkg", "c.py"), []byte("removed"), 0o644)

	diff, stats, err := generateDiff(oldDir, newDir)
	require.NoError(t, err)

	assert.Equal(t, 3, stats.filesChanged)
	assert.Greater(t, stats.linesAdded, 0)
	assert.Greater(t, stats.linesRemoved, 0)
	assert.Contains(t, diff, "a.py")
	assert.Contains(t, diff, "b.py")
	assert.Contains(t, diff, "c.py")
}

func TestGenerateDiff_BothEmpty(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	diff, stats, err := generateDiff(oldDir, newDir)
	require.NoError(t, err)
	assert.Empty(t, diff)
	assert.Equal(t, diffStats{}, stats)
}

func TestGenerateDiff_MultilineContent(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "pkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "pkg"), 0o755))

	oldContent := strings.Join([]string{"line1", "line2", "line3"}, "\n")
	newContent := strings.Join([]string{"line1", "modified", "line3", "line4"}, "\n")

	os.WriteFile(filepath.Join(oldDir, "pkg", "multi.txt"), []byte(oldContent), 0o644)
	os.WriteFile(filepath.Join(newDir, "pkg", "multi.txt"), []byte(newContent), 0o644)

	diff, stats, err := generateDiff(oldDir, newDir)
	require.NoError(t, err)

	assert.Equal(t, 1, stats.filesChanged)
	assert.Equal(t, 3, stats.linesRemoved)
	assert.Equal(t, 4, stats.linesAdded)
	assert.Contains(t, diff, "-line2")
	assert.Contains(t, diff, "+modified")
}

func TestNew_DefaultConfig(t *testing.T) {
	d := New(nil, nil, nil, Config{}, nil)
	assert.Equal(t, 100*1024, d.config.DiffSizeLimit)
}

func TestNew_CustomConfig(t *testing.T) {
	d := New(nil, nil, nil, Config{DiffSizeLimit: 50000}, nil)
	assert.Equal(t, 50000, d.config.DiffSizeLimit)
}

func TestGetEcosystem(t *testing.T) {
	d := New(nil, nil, nil, Config{}, nil)
	assert.Nil(t, d.getRegistry("unknown"))
	assert.Nil(t, d.getRegistry("python"))
	assert.Nil(t, d.getRegistry("npm"))
}
