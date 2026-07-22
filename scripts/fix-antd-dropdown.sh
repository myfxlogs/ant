#!/bin/bash
# Post-build patch: fix antd Dropdown to handle array children in React 19
# React 19's React.Children.only() throws on arrays.
# The isPrimitive check in Dropdown doesn't catch arrays.
# This patch adds Array.isArray() check.

set -euo pipefail

DIST_DIR="frontend/dist/assets"
TARGET_FILE=$(ls "$DIST_DIR"/vendor-antd-*.js 2>/dev/null | head -1)

if [ -z "$TARGET_FILE" ]; then
  echo "ERROR: vendor-antd bundle not found in $DIST_DIR"
  exit 1
fi

echo "Patching: $TARGET_FILE"

# Replace isPrimitive condition to also check for arrays
# Original: "object"!=typeof(G=i)&&!al(G)||null===G
# Patched:   ("object"!=typeof(G=i)&&!al(G)||null===G)||Array.isArray(i)
sed -i 's/"object"!=typeof(G=i)&&!al(G)||null===G/("object"!=typeof(G=i)\&\&!al(G)||null===G)||Array.isArray(i)/g' "$TARGET_FILE"

echo "Done. Patched Dropdown isPrimitive to handle arrays."
