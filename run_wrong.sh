#!/usr/bin/env bash

# Run the compiler against all wrong_*.ctds programs and summarize results.

set -u
set -o pipefail

PROJECT_ROOT=$(cd "$(dirname "$0")" && pwd)
SRC_DIR="$PROJECT_ROOT/target_source"

shopt -s nullglob
wrong_files=("$SRC_DIR"/wrong_*.ctds)
shopt -u nullglob

if (( ${#wrong_files[@]} == 0 )); then
  echo "No files matching $SRC_DIR/wrong_*.ctds"
  exit 1
fi

echo "Running compiler on ${#wrong_files[@]} wrong sample(s)..."

# Build compiler once into a temporary directory to run from ephemeral workdirs
BUILD_TMP=$(mktemp -d)
BIN_PATH="$BUILD_TMP/ctds"
if ! ( cd "$PROJECT_ROOT" && go build -o "$BIN_PATH" ); then
  echo "Failed to build compiler" >&2
  rm -rf "$BUILD_TMP"
  exit 1
fi
echo

total=0
pass=0   # expected failures (non-zero exit)
fail=0   # unexpected successes (zero exit)

printf "%s\n" "Case summary (PASS means compiler failed as expected):"
printf "%s\n" "------------------------------------------------------"

for file in "${wrong_files[@]}"; do
  (( total++ ))
  base=$(basename "$file" .ctds)

  # Run from project root so outputs are generated in result/
  cmd_output=$( ( cd "$PROJECT_ROOT" && "$BIN_PATH" "$file" ) 2>&1 )
  status=$?
  msg=$(printf '%s\n' "$cmd_output" | head -n 1)

  if (( status != 0 )); then
    (( pass++ ))
    printf "[PASS] %s (exit=%d) - %s\n" "$base" "$status" "$msg"
  else
    (( fail++ ))
    printf "[FAIL] %s (exit=%d) - %s\n" "$base" "$status" "$msg"
  fi
done

echo
echo "Summary:"
echo "  Total:      $total"
echo "  PASS (ok):  $pass"
echo "  FAIL (bad): $fail"
echo

# Cleanup build dir
rm -rf "$BUILD_TMP"

exit 0


