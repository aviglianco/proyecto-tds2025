#!/usr/bin/env bash

# Script that allows running all wrong_*.ctds programs to test that 
# the compiler fails as expected.
# Returns the results and error messages in a readable format.

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

BUILD_TMP=$(mktemp -d)
BIN_PATH="$BUILD_TMP/ctds"
if ! ( cd "$PROJECT_ROOT" && go build -o "$BIN_PATH" ); then
  echo "Failed to build compiler" >&2
  rm -rf "$BUILD_TMP"
  exit 1
fi
echo

total=0
pass=0
fail=0

printf "%s\n" "Case summary (PASS means compiler failed as expected):"
printf "%s\n" "------------------------------------------------------"

for file in "${wrong_files[@]}"; do
  (( total++ ))
  base=$(basename "$file" .ctds)

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

rm -rf "$BUILD_TMP"

exit 0


