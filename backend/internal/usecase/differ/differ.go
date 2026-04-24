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

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// Config holds configuration for the differ.
type Config struct {
	DiffSizeLimit int // Maximum diff size in bytes for LLM analysis
}

// Differ generates diffs between consecutive package releases.
type Differ struct {
	repo     DifferRepository
	python   usecase.Registry
	npm      usecase.Registry
	config   Config
	queue    usecase.QueueEnqueuer
	notifier usecase.NotificationDispatcher
}

// New creates a new Differ instance.
func New(repo DifferRepository, python usecase.Registry, npm usecase.Registry, config Config, q usecase.QueueEnqueuer, notifier usecase.NotificationDispatcher) *Differ {
	if config.DiffSizeLimit <= 0 {
		config.DiffSizeLimit = 100 * 1024 // 100KB default
	}
	return &Differ{
		repo:     repo,
		python:   python,
		npm:      npm,
		config:   config,
		queue:    q,
		notifier: notifier,
	}
}

// ProcessRelease generates a diff for a new release by its ID.
// This is the entry point for the queue worker.
func (d *Differ) ProcessRelease(ctx context.Context, releaseID string) error {
	release, pkg, err := d.repo.FindReleaseByIDWithPackage(ctx, releaseID)
	if err != nil {
		return fmt.Errorf("loading release %s: %w", releaseID, err)
	}

	// Update status to diffing
	d.repo.UpdateReleaseStatus(ctx, release.ID, entity.ReleaseStatusDiffing)

	// Find the previous release for this package by publish time
	prevRelease, err := d.repo.FindPreviousCompletedRelease(ctx, release.PackageID, release.PublishedAt)
	if err != nil || prevRelease == nil {
		// No previous release - mark as completed (first tracked version)
		slog.Info("no previous release found, skipping diff", "package_id", release.PackageID, "version", release.Version)
		d.repo.UpdateReleaseStatus(ctx, release.ID, entity.ReleaseStatusCompleted)
		return nil
	}

	// Get the appropriate registry client
	reg := d.getRegistry(string(pkg.Ecosystem))
	if reg == nil {
		return fmt.Errorf("unknown ecosystem: %s", pkg.Ecosystem)
	}

	// Download both tarballs
	newPath, err := reg.DownloadTarball(ctx, release.TarballURL)
	if err != nil {
		d.markError(ctx, release, pkg, "downloading new tarball: "+err.Error())
		return fmt.Errorf("downloading new tarball: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(newPath))

	oldPath, err := reg.DownloadTarball(ctx, prevRelease.TarballURL)
	if err != nil {
		d.markError(ctx, release, pkg, "downloading previous tarball: "+err.Error())
		return fmt.Errorf("downloading previous tarball: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(oldPath))

	// Extract tarballs
	newDir, err := extractTarball(newPath)
	if err != nil {
		d.markError(ctx, release, pkg, "extracting new tarball: "+err.Error())
		return fmt.Errorf("extracting new tarball: %w", err)
	}
	defer os.RemoveAll(newDir)

	oldDir, err := extractTarball(oldPath)
	if err != nil {
		d.markError(ctx, release, pkg, "extracting previous tarball: "+err.Error())
		return fmt.Errorf("extracting previous tarball: %w", err)
	}
	defer os.RemoveAll(oldDir)

	// Generate unified diff
	diffContent, stats, err := generateDiff(oldDir, newDir)
	if err != nil {
		d.markError(ctx, release, pkg, "generating diff: "+err.Error())
		return fmt.Errorf("generating diff: %w", err)
	}

	// Truncate diff if too large
	truncated := false
	originalSize := len(diffContent)
	if originalSize > d.config.DiffSizeLimit {
		diffContent = diffContent[:d.config.DiffSizeLimit] + "\n\n--- DIFF TRUNCATED (exceeded 100KB limit) ---\n"
		truncated = true
		slog.Warn("diff truncated",
			"package", pkg.Name,
			"version", release.Version,
			"original_size", originalSize,
			"limit", d.config.DiffSizeLimit,
		)
	}

	// Store diff
	diff := &entity.Diff{
		ReleaseID:        release.ID,
		PrevReleaseID:    prevRelease.ID,
		DiffContent:      diffContent,
		FileChangesCount: stats.filesChanged,
		LinesAdded:       stats.linesAdded,
		LinesRemoved:     stats.linesRemoved,
		Truncated:        truncated,
		OriginalSize:     originalSize,
	}

	if err := d.repo.CreateDiff(ctx, diff); err != nil {
		d.markError(ctx, release, pkg, "saving diff: "+err.Error())
		return fmt.Errorf("saving diff: %w", err)
	}

	// Update release status
	d.repo.UpdateReleaseStatus(ctx, release.ID, entity.ReleaseStatusAnalyzing)

	slog.Info("diff generated",
		"package", pkg.Name,
		"version", release.Version,
		"files_changed", stats.filesChanged,
		"lines_added", stats.linesAdded,
		"lines_removed", stats.linesRemoved,
	)

	// Enqueue analysis job via queue
	jobID, err := d.queue.Enqueue(ctx, "analyze", pkg.WorkspaceID, diff.ID)
	if err != nil {
		return fmt.Errorf("enqueuing analyze job: %w", err)
	}

	slog.Info("analysis job enqueued", "job_id", jobID, "diff_id", diff.ID)

	return nil
}

func (d *Differ) getRegistry(name string) usecase.Registry {
	switch name {
	case "python":
		return d.python
	case "npm":
		return d.npm
	default:
		return nil
	}
}

func (d *Differ) markError(ctx context.Context, release *entity.Release, pkg *entity.Package, msg string) {
	d.repo.UpdateReleaseError(ctx, release.ID, entity.ReleaseStatusError, msg)
	if d.notifier != nil && pkg.ID != "" {
		d.notifier.DispatchEvent(ctx, pkg.WorkspaceID, entity.NotificationEvent{
			Severity:      "medium",
			EventType:     entity.NotifEventDiffError,
			Title:         fmt.Sprintf("Diff failed: %s v%s", pkg.Name, release.Version),
			Message:       fmt.Sprintf("Failed to generate diff for %s v%s (%s): %s", pkg.Name, release.Version, pkg.Ecosystem, msg),
			ReferenceID:   release.ID,
			ReferenceType: "release",
		})
	}
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

			hunks := computeUnifiedHunks(oldLines, newLines, 3)
			for _, h := range hunks {
				diffBuilder.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", h.oldStart+1, h.oldCount, h.newStart+1, h.newCount))
				for _, op := range h.lines {
					diffBuilder.WriteString(op.text)
					diffBuilder.WriteString("\n")
					switch op.kind {
					case opAdd:
						stats.linesAdded++
					case opDel:
						stats.linesRemoved++
					}
				}
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

// --- Unified diff generation using Myers-like LCS approach (stdlib only) ---

const (
	opCtx = iota // context line (unchanged)
	opAdd        // added line
	opDel        // deleted line
)

type diffLine struct {
	kind int
	text string // includes prefix: " ", "+", or "-"
}

type hunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []diffLine
}

// computeUnifiedHunks computes a unified diff between old and new line slices,
// grouping changes into hunks with the given number of context lines.
func computeUnifiedHunks(oldLines, newLines []string, contextLines int) []hunk {
	// Compute edit script via LCS
	ops := computeEditScript(oldLines, newLines)

	if len(ops) == 0 {
		return nil
	}

	// Group into hunks
	var hunks []hunk
	var current *hunk
	flushDistance := contextLines*2 + 1

	oldIdx, newIdx := 0, 0

	for i, op := range ops {
		switch op.kind {
		case opCtx:
			// Check if we're near a change
			nearChange := false
			// Look backward: are we within contextLines of a previous change?
			for j := i - 1; j >= 0 && i-j <= contextLines; j-- {
				if ops[j].kind != opCtx {
					nearChange = true
					break
				}
			}
			// Look forward: are we within contextLines of a next change?
			if !nearChange {
				for j := i + 1; j < len(ops) && j-i <= contextLines; j++ {
					if ops[j].kind != opCtx {
						nearChange = true
						break
					}
				}
			}

			if nearChange {
				if current == nil {
					current = &hunk{oldStart: oldIdx, newStart: newIdx}
				}
				current.lines = append(current.lines, diffLine{kind: opCtx, text: " " + op.line})
				current.oldCount++
				current.newCount++
			} else if current != nil {
				// Check if next change is close enough to merge
				nextChangeDist := 0
				for j := i + 1; j < len(ops); j++ {
					if ops[j].kind != opCtx {
						nextChangeDist = j - i
						break
					}
				}
				if nextChangeDist > 0 && nextChangeDist <= flushDistance {
					current.lines = append(current.lines, diffLine{kind: opCtx, text: " " + op.line})
					current.oldCount++
					current.newCount++
				} else {
					hunks = append(hunks, *current)
					current = nil
				}
			}

			oldIdx++
			newIdx++
		case opDel:
			if current == nil {
				// Start new hunk, include preceding context
				start := oldIdx
				startNew := newIdx
				var ctx []diffLine
				for j := i - 1; j >= 0 && len(ctx) < contextLines; j-- {
					if ops[j].kind == opCtx {
						ctx = append([]diffLine{{kind: opCtx, text: " " + ops[j].line}}, ctx...)
						start--
						startNew--
					} else {
						break
					}
				}
				current = &hunk{oldStart: start, newStart: startNew, lines: ctx, oldCount: len(ctx), newCount: len(ctx)}
			}
			current.lines = append(current.lines, diffLine{kind: opDel, text: "-" + op.line})
			current.oldCount++
			oldIdx++
		case opAdd:
			if current == nil {
				start := oldIdx
				startNew := newIdx
				var ctx []diffLine
				for j := i - 1; j >= 0 && len(ctx) < contextLines; j-- {
					if ops[j].kind == opCtx {
						ctx = append([]diffLine{{kind: opCtx, text: " " + ops[j].line}}, ctx...)
						start--
						startNew--
					} else {
						break
					}
				}
				current = &hunk{oldStart: start, newStart: startNew, lines: ctx, oldCount: len(ctx), newCount: len(ctx)}
			}
			current.lines = append(current.lines, diffLine{kind: opAdd, text: "+" + op.line})
			current.newCount++
			newIdx++
		}
	}

	if current != nil {
		hunks = append(hunks, *current)
	}

	return hunks
}

type editOp struct {
	kind int // opCtx, opAdd, opDel
	line string
}

// computeEditScript computes a minimal edit script between old and new using LCS.
func computeEditScript(oldLines, newLines []string) []editOp {
	m, n := len(oldLines), len(newLines)

	// Optimize: use O(min(m,n)) space LCS with full path recovery
	// Build LCS table (O(m*n) time and space - acceptable for package diffs)
	// For very large files, we cap and fall back to simple approach
	const maxCells = 10_000_000 // ~10M cells max
	if int64(m)*int64(n) > maxCells {
		return fallbackEditScript(oldLines, newLines)
	}

	// Standard LCS DP
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to produce edit script
	var ops []editOp
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			ops = append(ops, editOp{kind: opCtx, line: oldLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, editOp{kind: opAdd, line: newLines[j-1]})
			j--
		} else {
			ops = append(ops, editOp{kind: opDel, line: oldLines[i-1]})
			i--
		}
	}

	// Reverse to get forward order
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}

	return ops
}

// fallbackEditScript handles very large files by showing all removed then all added.
func fallbackEditScript(oldLines, newLines []string) []editOp {
	ops := make([]editOp, 0, len(oldLines)+len(newLines))
	for _, l := range oldLines {
		ops = append(ops, editOp{kind: opDel, line: l})
	}
	for _, l := range newLines {
		ops = append(ops, editOp{kind: opAdd, line: l})
	}
	return ops
}
