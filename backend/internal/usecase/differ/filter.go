package differ

import (
	"path/filepath"
	"sort"
	"strings"
)

// Tier represents the priority level of a file for diff inclusion.
type Tier int

const (
	Tier1 Tier = iota // Code, config, unknown - highest priority
	Tier2             // CI/config directories (.github) - after code if budget allows
	Tier3             // Documentation text (.md, .txt) - after Tier 2 if budget allows
	Tier4             // Non-executable - always excluded
)

// tier4Extensions are binary/non-code extensions always excluded.
var tier4Extensions = map[string]bool{
	// Binary media
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
	".webp": true, ".bmp": true, ".tiff": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".otf": true,
	".mp3": true, ".mp4": true, ".wav": true, ".avi": true, ".mov": true,
	// Binary documents
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true,
	// Source maps
	".map": true,
}

// tier4Basenames are file names (lowercase) always excluded.
var tier4Basenames = map[string]bool{
	// Licenses
	"license": true, "license.md": true, "license.txt": true,
	"license-mit": true, "license-apache": true,
	"licence": true, "licence.md": true,
	// Notices/authors
	"notice": true, "notice.md": true,
	"authors": true, "authors.md": true,
	"contributors": true, "contributors.md": true,
	// Changelogs
	"changelog.md": true, "changelog.txt": true,
	"changes.md": true, "changes.txt": true,
	"history.md": true, "history.txt": true,
	// Lockfiles
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true,
	"pipfile.lock": true, "poetry.lock": true, "composer.lock": true,
	"cargo.lock": true, "gemfile.lock": true,
	// Config/dotfiles
	".gitignore": true, ".gitattributes": true, ".editorconfig": true,
	".prettierrc": true, ".prettierignore": true, ".eslintignore": true,
	".npmignore": true, ".dockerignore": true,
}

// tier4Dirs are directory names that cause exclusion (case-sensitive).
var tier4Dirs = map[string]bool{
	"__pycache__":  true,
	"node_modules": true,
	".git":         true,
}

// tier2Dirs are CI/config directories - included after code if budget allows (case-sensitive).
var tier2Dirs = map[string]bool{
	".github": true,
}

// tier3Extensions are documentation extensions included if budget allows.
var tier3Extensions = map[string]bool{
	".md": true, ".txt": true, ".rst": true, ".adoc": true, ".log": true,
}

// ClassifyFile returns the tier for a file path.
func ClassifyFile(path string, ecosystem string) Tier {
	segments := strings.Split(path, "/")

	// Check directory components (case-sensitive)
	for _, seg := range segments {
		if tier4Dirs[seg] {
			return Tier4
		}
		if tier2Dirs[seg] {
			return Tier2
		}
	}

	// Check basename (case-insensitive) against Tier 4
	base := strings.ToLower(filepath.Base(path))
	if tier4Basenames[base] {
		return Tier4
	}

	// Check extension (case-insensitive) against Tier 4
	ext := strings.ToLower(filepath.Ext(path))
	if tier4Extensions[ext] {
		return Tier4
	}

	// Check extension against Tier 3 (docs)
	if tier3Extensions[ext] {
		return Tier3
	}

	// Everything else is Tier 1
	return Tier1
}

// PartitionPaths classifies paths into four sorted slices by tier.
func PartitionPaths(paths []string, ecosystem string) (tier1, tier2, tier3, tier4 []string) {
	for _, p := range paths {
		switch ClassifyFile(p, ecosystem) {
		case Tier1:
			tier1 = append(tier1, p)
		case Tier2:
			tier2 = append(tier2, p)
		case Tier3:
			tier3 = append(tier3, p)
		case Tier4:
			tier4 = append(tier4, p)
		}
	}
	sort.Strings(tier1)
	sort.Strings(tier2)
	sort.Strings(tier3)
	sort.Strings(tier4)
	return
}
