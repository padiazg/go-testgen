#!/bin/sh
set -eu

# install.sh — Installs go-testgen AI agent skills
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/padiazg/go-testgen/main/scripts/install.sh | bash
#   curl -fsSL ... | bash -s -- /path/to/target/skills
#   ./scripts/install.sh /path/to/target/skills

REPO="padiazg/go-testgen"
BRANCH="${BRANCH:-main}"
TARGET="${1:-$HOME/.agents/skills}"
BASE="https://raw.githubusercontent.com/$REPO/$BRANCH/skills"

echo "Installing go-testgen AI agent skills to $TARGET"

# closure-check-tests skill
mkdir -p "$TARGET/closure-check-tests"
curl -fsSL "$BASE/closure-check-tests/SKILL.md" -o "$TARGET/closure-check-tests/SKILL.md"
echo "  ✔ closure-check-tests"

# gen-test-cases skill
mkdir -p "$TARGET/gen-test-cases"
curl -fsSL "$BASE/gen-test-cases/SKILL.md" -o "$TARGET/gen-test-cases/SKILL.md"
echo "  ✔ gen-test-cases"

# AGENTS.md (namespaced to avoid collisions)
mkdir -p "$TARGET/go-testgen"
curl -fsSL "$BASE/AGENTS.md" -o "$TARGET/go-testgen/AGENTS.md"
echo "  ✔ AGENTS.md"

echo "Done. Skills installed to $TARGET"
