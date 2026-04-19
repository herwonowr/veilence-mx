#!/usr/bin/env bash
# scan-secrets.sh - Scan the codebase for potential hardcoded secrets.
#
# Usage: ./scripts/scan-secrets.sh [path]
#
# Scans Go and TypeScript/JavaScript source files for patterns that may
# indicate hardcoded secrets, passwords, API keys, or tokens.
# Returns exit code 1 if any potential secrets are found.
#
# This script is designed to run in CI to prevent accidental secret commits.

set -euo pipefail

SCAN_PATH="${1:-.}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Scanning for hardcoded secrets in: ${SCAN_PATH}"
echo "=========================================="

FOUND=0

# Patterns to search for (case-insensitive)
# Each pattern is a regex paired with a description
declare -A PATTERNS
PATTERNS=(
  # API keys and tokens
  ["(api[_-]?key|apikey)\s*[:=]\s*['\"][a-zA-Z0-9]{16,}"]="Hardcoded API key"
  ["(secret|token)\s*[:=]\s*['\"][a-zA-Z0-9+/=]{16,}"]="Hardcoded secret/token"
  # AWS
  ["AKIA[0-9A-Z]{16}"]="AWS Access Key ID"
  ["aws[_-]?(secret|key)\s*[:=]\s*['\"]"]="AWS Secret reference"
  # Passwords
  ["password\s*[:=]\s*['\"][^'\"]{4,}['\"]"]="Hardcoded password"
  # Private keys
  ["-----BEGIN (RSA |EC |DSA )?PRIVATE KEY"]="Private key in source"
  # Connection strings with passwords
  ["(postgres|mysql|mongodb)://[^:]+:[^@]+@"]="Database connection string with password"
  # JWT secrets in source (not .env or test files)
  ["jwt[_-]?secret\s*[:=]\s*['\"][^'\"]{8,}"]="Hardcoded JWT secret"
)

# Files to exclude from scanning
EXCLUDE_PATTERNS=(
  "*.test.go"
  "*_test.go"
  "*.test.ts"
  "*.test.tsx"
  "*.spec.ts"
  "*.spec.tsx"
  ".env"
  ".env.*"
  "*.example"
  "go.sum"
  "package-lock.json"
  "node_modules/*"
  "vendor/*"
  ".git/*"
  "scripts/scan-secrets.sh"  # Don't flag ourselves
)

# Build grep exclude arguments
EXCLUDE_ARGS=""
for pattern in "${EXCLUDE_PATTERNS[@]}"; do
  EXCLUDE_ARGS="${EXCLUDE_ARGS} --exclude=${pattern}"
done
EXCLUDE_ARGS="${EXCLUDE_ARGS} --exclude-dir=node_modules --exclude-dir=vendor --exclude-dir=.git --exclude-dir=_workspace"

for pattern in "${!PATTERNS[@]}"; do
  description="${PATTERNS[$pattern]}"
  # Use grep with extended regex, case-insensitive
  results=$(grep -rniE ${EXCLUDE_ARGS} "${pattern}" "${SCAN_PATH}" 2>/dev/null || true)
  if [ -n "$results" ]; then
    echo -e "${RED}POTENTIAL SECRET: ${description}${NC}"
    echo "$results" | head -20
    echo ""
    FOUND=$((FOUND + 1))
  fi
done

echo "=========================================="
if [ $FOUND -gt 0 ]; then
  echo -e "${RED}Found ${FOUND} potential secret pattern(s)!${NC}"
  echo -e "${YELLOW}Review each finding above. False positives can be excluded in scan-secrets.sh.${NC}"
  exit 1
else
  echo -e "${GREEN}No hardcoded secrets detected.${NC}"
  exit 0
fi
