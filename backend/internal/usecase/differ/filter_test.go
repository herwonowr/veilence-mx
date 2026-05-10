package differ

import (
	"testing"
)

func TestClassifyFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want Tier
	}{
		// Tier 4 extensions
		{"png image", "icon.png", Tier4},
		{"pdf doc", "data.pdf", Tier4},
		{"source map", "bundle.map", Tier4},
		{"jpeg image", "photo.jpeg", Tier4},
		{"font woff2", "font.woff2", Tier4},
		{"video mp4", "video.mp4", Tier4},

		// Tier 4 basenames
		{"LICENSE", "LICENSE", Tier4},
		{"LICENSE.md", "LICENSE.md", Tier4},
		{"CHANGELOG.md", "CHANGELOG.md", Tier4},
		{"package-lock.json", "package-lock.json", Tier4},
		{".gitignore", ".gitignore", Tier4},
		{"yarn.lock", "yarn.lock", Tier4},
		{"poetry.lock", "poetry.lock", Tier4},
		{".editorconfig", ".editorconfig", Tier4},
		{".prettierrc", ".prettierrc", Tier4},
		{".golangci.yml", ".golangci.yml", Tier4},
		{".eslintrc.json", ".eslintrc.json", Tier4},
		{".travis.yml", ".travis.yml", Tier4},
		{".nvmrc", ".nvmrc", Tier4},
		{".gitpod.yml", ".gitpod.yml", Tier4},
		{"codecov.yml", "codecov.yml", Tier4},
		{"renovate.json", "renovate.json", Tier4},
		{".coveragerc", ".coveragerc", Tier4},
		{".babelrc", ".babelrc", Tier4},

		// Tier 4 directories
		{"pycache root", "__pycache__/foo.pyc", Tier4},
		{"pycache nested", "src/__pycache__/foo.pyc", Tier4},
		{"node_modules", "node_modules/lodash/index.js", Tier4},
		{".git dir", ".git/config", Tier4},

		// Tier 2 - CI/config directories
		{".github workflow", ".github/workflows/ci.yml", Tier2},
		{".github issue template", ".github/ISSUE_TEMPLATE/bug.md", Tier2},
		{".github dependabot", ".github/dependabot.yml", Tier2},
		{".github actions", ".github/actions/setup/action.yml", Tier2},
		{".circleci config", ".circleci/config.yml", Tier2},
		{".gitlab ci", ".gitlab/ci/build.yml", Tier2},
		{".travis yml", ".travis/deploy.sh", Tier2},
		{".buildkite pipeline", ".buildkite/pipeline.yml", Tier2},
		{".jenkins pipeline", ".jenkins/Jenkinsfile", Tier2},
		{".gitea workflow", ".gitea/workflows/ci.yml", Tier2},
		{".forgejo workflow", ".forgejo/workflows/build.yml", Tier2},
		{".woodpecker pipeline", ".woodpecker/build.yml", Tier2},
		{".drone pipeline", ".drone/config.yml", Tier2},

		// Tier 3 docs
		{"README.md", "README.md", Tier3},
		{"docs guide", "docs/guide.md", Tier3},
		{"notes.txt", "notes.txt", Tier3},
		{"api rst", "docs/api.rst", Tier3},
		{"adoc file", "docs/intro.adoc", Tier3},
		{"log file", "server.log", Tier3},

		// Tier 1 code
		{"python", "main.py", Tier1},
		{"javascript", "src/index.js", Tier1},
		{"setup.py", "setup.py", Tier1},
		{"package.json", "package.json", Tier1},
		{"go.mod", "go.mod", Tier1},
		{"go.sum", "go.sum", Tier1},

		// Tier 1 - NOT excluded dirs
		{"dist js", "dist/main.js", Tier1},
		{"build output", "build/output.js", Tier1},
		{"vendor lib", "vendor/lib/lib.go", Tier1},
		{"svg file", "image.svg", Tier1},

		// Tier 1 - unknown/suspicious
		{"unknown ext", "payload.xyz", Tier1},
		{"wasm", "script.wasm", Tier1},
		{"no extension", "noextension", Tier1},
		{"ruby file", "data.rb", Tier1},

		// Case insensitivity
		{"license lowercase ext", "license.MD", Tier4},
		{"changelog txt upper", "CHANGELOG.TXT", Tier4},
		{"readme upper ext", "README.MD", Tier3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyFile(tt.path, "npm")
			if got != tt.want {
				t.Errorf("ClassifyFile(%q) = %d, want %d", tt.path, got, tt.want)
			}
		})
	}
}

func TestPartitionPaths(t *testing.T) {
	paths := []string{
		"src/index.js",
		"README.md",
		"LICENSE",
		"icon.png",
		"main.py",
		"docs/guide.md",
		"package-lock.json",
		"setup.py",
		"__pycache__/foo.pyc",
		".github/workflows/ci.yml",
		".github/dependabot.yml",
	}

	tier1, tier2, tier3, tier4 := PartitionPaths(paths, "npm")

	// Check tier1 (code)
	expectedT1 := []string{"main.py", "setup.py", "src/index.js"}
	if len(tier1) != len(expectedT1) {
		t.Fatalf("tier1 length = %d, want %d", len(tier1), len(expectedT1))
	}
	for i, v := range expectedT1 {
		if tier1[i] != v {
			t.Errorf("tier1[%d] = %q, want %q", i, tier1[i], v)
		}
	}

	// Check tier2 (CI/config dirs)
	expectedT2 := []string{".github/dependabot.yml", ".github/workflows/ci.yml"}
	if len(tier2) != len(expectedT2) {
		t.Fatalf("tier2 length = %d, want %d", len(tier2), len(expectedT2))
	}
	for i, v := range expectedT2 {
		if tier2[i] != v {
			t.Errorf("tier2[%d] = %q, want %q", i, tier2[i], v)
		}
	}

	// Check tier3 (docs)
	expectedT3 := []string{"README.md", "docs/guide.md"}
	if len(tier3) != len(expectedT3) {
		t.Fatalf("tier3 length = %d, want %d", len(tier3), len(expectedT3))
	}
	for i, v := range expectedT3 {
		if tier3[i] != v {
			t.Errorf("tier3[%d] = %q, want %q", i, tier3[i], v)
		}
	}

	// Check tier4 (excluded)
	expectedT4 := []string{"LICENSE", "__pycache__/foo.pyc", "icon.png", "package-lock.json"}
	if len(tier4) != len(expectedT4) {
		t.Fatalf("tier4 length = %d, want %d", len(tier4), len(expectedT4))
	}
	for i, v := range expectedT4 {
		if tier4[i] != v {
			t.Errorf("tier4[%d] = %q, want %q", i, tier4[i], v)
		}
	}

	// Verify all tiers are sorted
	for i := 1; i < len(tier1); i++ {
		if tier1[i] < tier1[i-1] {
			t.Errorf("tier1 not sorted at index %d", i)
		}
	}
	for i := 1; i < len(tier2); i++ {
		if tier2[i] < tier2[i-1] {
			t.Errorf("tier2 not sorted at index %d", i)
		}
	}
	for i := 1; i < len(tier3); i++ {
		if tier3[i] < tier3[i-1] {
			t.Errorf("tier3 not sorted at index %d", i)
		}
	}
	for i := 1; i < len(tier4); i++ {
		if tier4[i] < tier4[i-1] {
			t.Errorf("tier4 not sorted at index %d", i)
		}
	}
}
