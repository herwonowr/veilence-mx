package differ

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkGenerateDiff_SmallFiles measures diff generation with a few small files.
// Baseline: should complete in <1ms/op for typical package updates.
func BenchmarkGenerateDiff_SmallFiles(b *testing.B) {
	oldDir := b.TempDir()
	newDir := b.TempDir()

	setupSmallDiffDirs(b, oldDir, newDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		diff, stats, err := generateDiff(oldDir, newDir)
		if err != nil {
			b.Fatal(err)
		}
		if stats.filesChanged == 0 {
			b.Fatal("expected changes")
		}
		if len(diff) == 0 {
			b.Fatal("expected non-empty diff")
		}
	}
}

// BenchmarkGenerateDiff_MediumProject measures diff generation for a medium-sized project (~50 files).
func BenchmarkGenerateDiff_MediumProject(b *testing.B) {
	oldDir := b.TempDir()
	newDir := b.TempDir()

	setupMediumDiffDirs(b, oldDir, newDir, 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, stats, err := generateDiff(oldDir, newDir)
		if err != nil {
			b.Fatal(err)
		}
		if stats.filesChanged == 0 {
			b.Fatal("expected changes")
		}
	}
}

// BenchmarkGenerateDiff_LargeProject measures diff generation for a large project (~200 files).
func BenchmarkGenerateDiff_LargeProject(b *testing.B) {
	oldDir := b.TempDir()
	newDir := b.TempDir()

	setupMediumDiffDirs(b, oldDir, newDir, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, stats, err := generateDiff(oldDir, newDir)
		if err != nil {
			b.Fatal(err)
		}
		if stats.filesChanged == 0 {
			b.Fatal("expected changes")
		}
	}
}

// BenchmarkGenerateDiff_NoChanges measures diff cost when nothing changed.
func BenchmarkGenerateDiff_NoChanges(b *testing.B) {
	oldDir := b.TempDir()
	newDir := b.TempDir()

	setupIdenticalDirs(b, oldDir, newDir, 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		diff, _, err := generateDiff(oldDir, newDir)
		if err != nil {
			b.Fatal(err)
		}
		if len(diff) != 0 {
			b.Fatal("expected empty diff")
		}
	}
}

// BenchmarkWalkFiles measures file tree walking for typical package size.
func BenchmarkWalkFiles(b *testing.B) {
	dir := b.TempDir()
	setupFileTree(b, dir, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := walkFiles(dir)
		if err != nil {
			b.Fatal(err)
		}
		if len(files) == 0 {
			b.Fatal("expected files")
		}
	}
}

// BenchmarkExtractTarball measures tarball extraction throughput.
func BenchmarkExtractTarball(b *testing.B) {
	files := make(map[string]string)
	for j := 0; j < 20; j++ {
		name := fmt.Sprintf("pkg-1.0/file_%d.py", j)
		content := strings.Repeat(fmt.Sprintf("# File %d\nprint('hello from file %d')\n", j, j), 10)
		files[name] = content
	}

	// createTarball uses testing.T, so we create it manually for benchmarks
	tarPath := createBenchTarball(b, files)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dir, err := extractTarball(tarPath)
		if err != nil {
			b.Fatal(err)
		}
		os.RemoveAll(dir)
	}
}

// BenchmarkExtractTarball_Large measures tarball extraction for larger packages.
func BenchmarkExtractTarball_Large(b *testing.B) {
	files := make(map[string]string)
	for j := 0; j < 100; j++ {
		name := fmt.Sprintf("pkg-2.0/src/module_%d.py", j)
		content := strings.Repeat(fmt.Sprintf("class Module%d:\n    def run(self):\n        return %d\n", j, j), 20)
		files[name] = content
	}

	tarPath := createBenchTarball(b, files)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dir, err := extractTarball(tarPath)
		if err != nil {
			b.Fatal(err)
		}
		os.RemoveAll(dir)
	}
}

// --- Helpers ---

func setupSmallDiffDirs(b *testing.B, oldDir, newDir string) {
	b.Helper()

	pkg := filepath.Join(oldDir, "pkg")
	os.MkdirAll(pkg, 0o755)
	os.WriteFile(filepath.Join(pkg, "setup.py"), []byte("from setuptools import setup\nsetup(name='pkg', version='1.0')\n"), 0o644)
	os.WriteFile(filepath.Join(pkg, "main.py"), []byte("def main():\n    print('hello')\n"), 0o644)
	os.WriteFile(filepath.Join(pkg, "utils.py"), []byte("def helper():\n    return 42\n"), 0o644)

	pkg2 := filepath.Join(newDir, "pkg")
	os.MkdirAll(pkg2, 0o755)
	os.WriteFile(filepath.Join(pkg2, "setup.py"), []byte("from setuptools import setup\nsetup(name='pkg', version='2.0')\n"), 0o644)
	os.WriteFile(filepath.Join(pkg2, "main.py"), []byte("def main():\n    print('hello world')\n    print('goodbye')\n"), 0o644)
	os.WriteFile(filepath.Join(pkg2, "utils.py"), []byte("def helper():\n    return 42\n"), 0o644) // unchanged
	os.WriteFile(filepath.Join(pkg2, "new_feature.py"), []byte("def feature():\n    return True\n"), 0o644)
}

func setupMediumDiffDirs(b *testing.B, oldDir, newDir string, fileCount int) {
	b.Helper()

	oldPkg := filepath.Join(oldDir, "pkg")
	newPkg := filepath.Join(newDir, "pkg")
	os.MkdirAll(oldPkg, 0o755)
	os.MkdirAll(newPkg, 0o755)

	for j := 0; j < fileCount; j++ {
		name := fmt.Sprintf("module_%d.py", j)
		oldContent := fmt.Sprintf("# Module %d v1\ndef func_%d():\n    return %d\n", j, j, j)
		os.WriteFile(filepath.Join(oldPkg, name), []byte(oldContent), 0o644)

		// 30% of files are modified, 10% are deleted in new, 10% are new
		switch {
		case j%10 == 0:
			// Deleted in new version (don't write to newDir)
		case j%3 == 0:
			// Modified
			newContent := fmt.Sprintf("# Module %d v2 (updated)\ndef func_%d():\n    return %d * 2\n", j, j, j)
			os.WriteFile(filepath.Join(newPkg, name), []byte(newContent), 0o644)
		default:
			// Unchanged
			os.WriteFile(filepath.Join(newPkg, name), []byte(oldContent), 0o644)
		}
	}

	// Add some new files
	for j := fileCount; j < fileCount+fileCount/10; j++ {
		name := fmt.Sprintf("new_module_%d.py", j)
		content := fmt.Sprintf("# New module %d\ndef new_func_%d():\n    pass\n", j, j)
		os.WriteFile(filepath.Join(newPkg, name), []byte(content), 0o644)
	}
}

func setupIdenticalDirs(b *testing.B, oldDir, newDir string, fileCount int) {
	b.Helper()

	oldPkg := filepath.Join(oldDir, "pkg")
	newPkg := filepath.Join(newDir, "pkg")
	os.MkdirAll(oldPkg, 0o755)
	os.MkdirAll(newPkg, 0o755)

	for j := 0; j < fileCount; j++ {
		name := fmt.Sprintf("module_%d.py", j)
		content := fmt.Sprintf("# Module %d\ndef func_%d():\n    return %d\n", j, j, j)
		os.WriteFile(filepath.Join(oldPkg, name), []byte(content), 0o644)
		os.WriteFile(filepath.Join(newPkg, name), []byte(content), 0o644)
	}
}

func setupFileTree(b *testing.B, dir string, fileCount int) {
	b.Helper()

	pkg := filepath.Join(dir, "pkg")
	os.MkdirAll(filepath.Join(pkg, "sub1"), 0o755)
	os.MkdirAll(filepath.Join(pkg, "sub2", "nested"), 0o755)

	for j := 0; j < fileCount; j++ {
		var name string
		switch {
		case j%3 == 0:
			name = filepath.Join(pkg, fmt.Sprintf("file_%d.py", j))
		case j%3 == 1:
			name = filepath.Join(pkg, "sub1", fmt.Sprintf("file_%d.py", j))
		default:
			name = filepath.Join(pkg, "sub2", "nested", fmt.Sprintf("file_%d.py", j))
		}
		content := fmt.Sprintf("# File %d\ndef func():\n    return %d\n", j, j)
		os.WriteFile(name, []byte(content), 0o644)
	}
}

func createBenchTarball(b *testing.B, files map[string]string) string {
	b.Helper()
	tmpDir := b.TempDir()
	tarPath := filepath.Join(tmpDir, "package.tar.gz")

	f, err := os.Create(tarPath)
	if err != nil {
		b.Fatal(err)
	}
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
		if err := tw.WriteHeader(hdr); err != nil {
			b.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			b.Fatal(err)
		}
	}

	return tarPath
}
