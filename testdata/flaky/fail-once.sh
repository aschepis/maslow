#!/bin/sh
# Fails the first time it runs and passes after, for as long as the marker file
# named by $1 exists. Delete the marker to make it fail once again.
marker="$1"
if [ -f "$marker" ]; then
  echo "marker found: pass"
  exit 0
fi
mkdir -p "$(dirname "$marker")"
touch "$marker"
echo "no marker yet: fail"
exit 1
