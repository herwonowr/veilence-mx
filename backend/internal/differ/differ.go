package differ

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/queue"
	"github.com/veilence/veilence-mx/backend/internal/registry"
	"gorm.io/gorm"
)

// Config holds configuration for the differ.
type Config struct {
	DiffSizeLimit int // Maximum diff size in bytes for LLM analysis
}

// Differ generates diffs between consecutive package releases.
type Differ struct {
	db     *gorm.DB
	python registry.Registry
	npm    registry.Registry
	config Config
	queue  *queue.Queue
}

// New creates a new Differ instance.
func New(db *gorm.DB, python registry.Registry, npm registry.Registry, config Config, q *queue.Queue) *Differ {
	if config.DiffSizeLimit <= 0 {
		config.DiffSizeLimit = 100 * 1024 // 100KB default
	}
	return &Differ{
		db:     db,
		python: python,
		npm:    npm,
		config: config,
		queue:  q,
	}
}

// ProcessJob is the queue worker handler for diff jobs.
func (d *Differ) ProcessJob(ctx context.Context, job *queue.Job) error {
	return d.processRelease(ctx, job.ReferenceID)
}

// processRelease generates a diff for a new release.
func (d *Differ) processRelease(ctx context.Context, releaseID uint) error {
	var release models.Release
	if err := d.db.Preload("Package").First(&release, releaseID).Error; err != nil {
		return fmt.Errorf("loading release %d: %w", releaseID, err)
	}

	// Update status to diffing
	d.db.Model(&release).Update("status", models.ReleaseStatusDiffing)

	// Find the previous release for this package by publish time
	var prevRelease models.Release
	result := d.db.Where("package_id = ? AND published_at < ? AND status = ?", release.PackageID, release.PublishedAt, models.ReleaseStatusCompleted).
		Order("published_at DESC").Limit(1).Find(&prevRelease)

	if result.RowsAffected == 0 {
		// No previous release — mark as completed (first tracked version)
		slog.Info("no previous release found, skipping diff", "package_id", release.PackageID, "version", release.Version)
		d.db.Model(&release).Update("status", models.ReleaseStatusCompleted)
		return nil
	}

	// Get the appropriate registry client
	reg := d.getRegistry(string(release.Package.Ecosystem))
	if reg == nil {
		return fmt.Errorf("unknown ecosystem: %s", release.Package.Ecosystem)
	}

	// Download both tarballs
	newPath, err := reg.DownloadTarball(ctx, release.TarballURL)
	if err != nil {
		d.markError(&release, "downloading new tarball: "+err.Error())
		return fmt.Errorf("downloading new tarball: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(newPath))

	oldPath, err := reg.DownloadTarball(ctx, prevRelease.TarballURL)
	if err != nil {
		d.markError(&release, "downloading previous tarball: "+err.Error())
		return fmt.Errorf("downloading previous tarball: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(oldPath))

	// Extract tarballs
	newDir, err := extractTarball(newPath)
	if err != nil {
		d.markError(&release, "extracting new tarball: "+err.Error())
		return fmt.Errorf("extracting new tarball: %w", err)
	}
	defer os.RemoveAll(newDir)

	oldDir, err := extractTarball(oldPath)
	if err != nil {
		d.markError(&release, "extracting previous tarball: "+err.Error())
		return fmt.Errorf("extracting previous tarball: %w", err)
	}
	defer os.RemoveAll(oldDir)

	// Generate unified diff
	diffContent, stats, err := generateDiff(oldDir, newDir)
	if err != nil {
		d.markError(&release, "generating diff: "+err.Error())
		return fmt.Errorf("generating diff: %w", err)
	}

	// Truncate diff if too large
	if len(diffContent) > d.config.DiffSizeLimit {
		diffContent = diffContent[:d.config.DiffSizeLimit] + "\n... [diff truncated]"
	}

	// Store diff
	diff := models.Diff{
		ReleaseID:        release.ID,
		PrevReleaseID:    prevRelease.ID,
		DiffContent:      diffContent,
		FileChangesCount: stats.filesChanged,
		LinesAdded:       stats.linesAdded,
		LinesRemoved:     stats.linesRemoved,
	}

	if err := d.db.Create(&diff).Error; err != nil {
		d.markError(&release, "saving diff: "+err.Error())
		return fmt.Errorf("saving diff: %w", err)
	}

	// Update release status
	d.db.Model(&release).Update("status", models.ReleaseStatusAnalyzing)

	slog.Info("diff generated",
		"package", release.Package.Name,
		"version", release.Version,
		"files_changed", stats.filesChanged,
		"lines_added", stats.linesAdded,
		"lines_removed", stats.linesRemoved,
	)

	// Enqueue analysis job via Redis queue
	jobID, err := d.queue.Enqueue(ctx, queue.JobTypeAnalyze, diff.ID)
	if err != nil {
		return fmt.Errorf("enqueuing analyze job: %w", err)
	}

	slog.Info("analysis job enqueued", "job_id", jobID, "diff_id", diff.ID)

	return nil
}

func (d *Differ) getRegistry(name string) registry.Registry {
	switch name {
	case "python":
		return d.python
	case "npm":
		return d.npm
	default:
		return nil
	}
}

func (d *Differ) markError(release *models.Release, msg string) {
	d.db.Model(release).Updates(map[string]any{
		"status":        models.ReleaseStatusError,
		"error_message": msg,
	})
}

type diffStats struct {
	filesChanged int
	linesAdded   int
	linesRemoved int
}

// extractTarball extracts a .tar.gz or .tgz file to a temp directory.
func extractTarball(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening tarball: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gz.Close()

	destDir, err := os.MkdirTemp("", "veilence-extract-*")
	if err != nil {
		return "", fmt.Errorf("creating extraction directory: %w", err)
	}

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading tar entry: %w", err)
		}

		// Sanitize path to prevent directory traversal
		cleanName := filepath.Clean(header.Name)
		if strings.Contains(cleanName, "..") {
			continue
		}

		target := filepath.Join(destDir, cleanName)

		// Ensure the target is within destDir
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o750); err != nil {
				return "", fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
				return "", fmt.Errorf("creating parent directory: %w", err)
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode)&0o750)
			if err != nil {
				return "", fmt.Errorf("creating file %s: %w", target, err)
			}
			if _, err := io.Copy(outFile, io.LimitReader(tr, 50*1024*1024)); err != nil {
				outFile.Close()
				return "", fmt.Errorf("extracting file %s: %w", target, err)
			}
			outFile.Close()
		}
	}

	return destDir, nil
}

// generateDiff generates a unified diff between two directories.
func generateDiff(oldDir, newDir string) (string, diffStats, error) {
	oldFiles, err := walkFiles(oldDir)
	if err != nil {
		return "", diffStats{}, fmt.Errorf("walking old directory: %w", err)
	}

	newFiles, err := walkFiles(newDir)
	if err != nil {
		return "", diffStats{}, fmt.Errorf("walking new directory: %w", err)
	}

	// Merge all file paths
	allPaths := make(map[string]bool)
	for p := range oldFiles {
		allPaths[p] = true
	}
	for p := range newFiles {
		allPaths[p] = true
	}

	sortedPaths := make([]string, 0, len(allPaths))
	for p := range allPaths {
		sortedPaths = append(sortedPaths, p)
	}
	sort.Strings(sortedPaths)

	var diffBuilder strings.Builder
	stats := diffStats{}

	for _, path := range sortedPaths {
		oldContent, oldExists := oldFiles[path]
		newContent, newExists := newFiles[path]

		if oldContent == newContent {
			continue
		}

		stats.filesChanged++

		if !oldExists {
			diffBuilder.WriteString(fmt.Sprintf("--- /dev/null\n+++ b/%s\n", path))
			lines := strings.Split(newContent, "\n")
			stats.linesAdded += len(lines)
			for _, line := range lines {
				diffBuilder.WriteString("+" + line + "\n")
			}
		} else if !newExists {
			diffBuilder.WriteString(fmt.Sprintf("--- a/%s\n+++ /dev/null\n", path))
			lines := strings.Split(oldContent, "\n")
			stats.linesRemoved += len(lines)
			for _, line := range lines {
				diffBuilder.WriteString("-" + line + "\n")
			}
		} else {
			diffBuilder.WriteString(fmt.Sprintf("--- a/%s\n+++ b/%s\n", path, path))
			oldLines := strings.Split(oldContent, "\n")
			newLines := strings.Split(newContent, "\n")

			// Simple line-by-line diff (for proper unified diff, use a diff library)
			for _, line := range oldLines {
				diffBuilder.WriteString("-" + line + "\n")
				stats.linesRemoved++
			}
			for _, line := range newLines {
				diffBuilder.WriteString("+" + line + "\n")
				stats.linesAdded++
			}
		}
		diffBuilder.WriteString("\n")
	}

	return diffBuilder.String(), stats, nil
}

// walkFiles reads all files in a directory tree and returns their contents.
func walkFiles(root string) (map[string]string, error) {
	files := make(map[string]string)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Skip binary files (simple heuristic: skip files > 1MB)
		if info.Size() > 1024*1024 {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Strip the first directory component (package name + version prefix)
		parts := strings.SplitN(relPath, string(os.PathSeparator), 2)
		if len(parts) > 1 {
			relPath = parts[1]
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		files[relPath] = string(content)
		return nil
	})

	return files, err
}
