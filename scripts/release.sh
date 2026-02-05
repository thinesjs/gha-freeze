#!/usr/bin/env bash
set -e

BUMP_TYPE="${1:-auto}"

if [[ ! "$BUMP_TYPE" =~ ^(auto|patch|minor|major)$ ]]; then
  echo "Usage: $0 [auto|patch|minor|major]"
  echo ""
  echo "  auto   - Automatically detect bump type from commits (default)"
  echo "  patch  - Bump patch version (0.0.x) for bug fixes"
  echo "  minor  - Bump minor version (0.x.0) for new features"
  echo "  major  - Bump major version (x.0.0) for breaking changes"
  exit 1
fi

git fetch --tags

LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "📌 Latest tag: $LATEST_TAG"

VERSION=${LATEST_TAG#v}
IFS='.' read -r -a VERSION_PARTS <<< "$VERSION"
MAJOR="${VERSION_PARTS[0]:-0}"
MINOR="${VERSION_PARTS[1]:-0}"
PATCH="${VERSION_PARTS[2]:-0}"

if [ "$BUMP_TYPE" = "auto" ]; then
  COMMITS=$(git log ${LATEST_TAG}..HEAD --pretty=format:"%s" 2>/dev/null || git log --pretty=format:"%s")

  if echo "$COMMITS" | grep -qE "^feat(\(.*\))?!:|^[a-z]+(\(.*\))?!:|BREAKING CHANGE:"; then
    BUMP_TYPE="major"
    echo "🔍 Auto-detected: MAJOR (breaking changes found)"
  elif echo "$COMMITS" | grep -qE "^feat(\(.*\))?:"; then
    BUMP_TYPE="minor"
    echo "🔍 Auto-detected: MINOR (new features found)"
  else
    BUMP_TYPE="patch"
    echo "🔍 Auto-detected: PATCH (fixes only)"
  fi
fi

case "$BUMP_TYPE" in
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  patch)
    PATCH=$((PATCH + 1))
    ;;
esac

NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
NEW_TAG="v${NEW_VERSION}"

echo ""
echo "📦 New version: $NEW_TAG (${BUMP_TYPE} bump)"
echo ""
echo "📝 Changes since $LATEST_TAG:"
git log ${LATEST_TAG}..HEAD --pretty=format:"  • %s (%h)" --no-merges 2>/dev/null || \
  git log --pretty=format:"  • %s (%h)" --no-merges
echo ""
echo ""

read -p "🚀 Create and push tag $NEW_TAG? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo "❌ Release cancelled"
  exit 1
fi

git tag -a "$NEW_TAG" -m "Release $NEW_TAG"
git push origin "$NEW_TAG"

echo ""
echo "✅ Tag $NEW_TAG created and pushed"
echo "🔄 GitHub Actions will now build and publish the release"
echo "📦 View release at: https://github.com/thinesjs/gha-freeze/releases/tag/$NEW_TAG"
