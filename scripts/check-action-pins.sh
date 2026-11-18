#!/bin/sh
# Fails when a published Action uses another action by anything but a full
# commit SHA; callers pin our Action by SHA and must get exactly that code.
set -u

files="${*:-action/action.yml}"
unpinned=$(grep -nE '^[[:space:]]*(-[[:space:]]+)?uses:[[:space:]]+[^.[:space:]]' $files | grep -vE '@[0-9a-f]{40}([[:space:]]|$)')

if [ -n "$unpinned" ]; then
  echo "actions must be pinned to a full commit SHA:"
  echo "$unpinned"
  exit 1
fi
echo "all actions pinned"
