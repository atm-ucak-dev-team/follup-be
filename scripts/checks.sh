#!/bin/sh
# Run gofmt check and go test for staged .go files

echo "🔍 Running Go checks..."

# ── 1. Check if any .go files are staged ──────────────────────────────────────
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$')

if [ -z "$STAGED_GO_FILES" ]; then
  echo "   No staged .go files, skipping."
  exit 0
fi

# ── 2. gofmt check ────────────────────────────────────────────────────────────
echo ""
echo "📐 Checking gofmt..."

UNFORMATTED=""
for FILE in $STAGED_GO_FILES; do
  if [ -f "$FILE" ]; then
    DIFF=$(gofmt -l "$FILE")
    if [ -n "$DIFF" ]; then
      UNFORMATTED="$UNFORMATTED\n     $FILE"
    fi
  fi
done

if [ -n "$UNFORMATTED" ]; then
  echo ""
  echo "❌ gofmt issues found in:"
  printf "%b\n" "$UNFORMATTED"
  echo ""
  echo "   Fix with: gofmt -w ."
  echo ""
  exit 1
fi

echo "   ✅ gofmt OK"

# ── 3. go test ────────────────────────────────────────────────────────────────
echo ""
echo "🧪 Running go test ./..."
echo ""

go test ./...
TEST_EXIT=$?

if [ $TEST_EXIT -ne 0 ]; then
  echo ""
  echo "❌ Tests failed. Please fix before committing."
  echo ""
  exit 1
fi

echo ""
echo "✅ All Go checks passed!"
exit 0
