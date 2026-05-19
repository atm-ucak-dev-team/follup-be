#!/bin/sh
# Validate commit message prefix
# Valid prefixes: feat, fix, chore, docs

COMMIT_MSG_FILE=$1
COMMIT_MSG=$(cat "$COMMIT_MSG_FILE")

VALID_PREFIXES="^(feat|fix|chore|docs)(\(.+\))?: .+"

if ! echo "$COMMIT_MSG" | grep -qE "$VALID_PREFIXES"; then
  echo ""
  echo "❌ Invalid commit message format!"
  echo ""
  echo "   Your message: \"$COMMIT_MSG\""
  echo ""
  echo "   Required format: <prefix>: <description>"
  echo "   Or with scope:   <prefix>(<scope>): <description>"
  echo ""
  echo "   Valid prefixes:"
  echo "     feat     → new feature"
  echo "     fix      → bug fix"
  echo "     chore    → maintenance / tooling"
  echo "     docs     → documentation"
  echo ""
  echo "   Examples:"
  echo "     feat: add user login"
  echo "     fix(auth): handle token expiry"
  echo "     chore: update dependencies"
  echo "     docs: update README"
  echo ""
  exit 1
fi

echo "✅ Commit message valid."
exit 0
