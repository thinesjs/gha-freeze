#!/usr/bin/env bash
set -e

echo "🪝 Setting up git hooks..."

if [ ! -d .git ]; then
    echo "❌ Not a git repository. Run this from the repository root."
    exit 1
fi

git config core.hooksPath .githooks

echo "✅ Git hooks configured!"
echo ""
echo "Pre-commit hook will now run on every commit to check:"
echo "  • Code formatting (gofmt)"
echo "  • Static analysis (go vet)"
echo "  • Linting (golangci-lint)"
echo ""
echo "To skip hooks for a specific commit, use: git commit --no-verify"
