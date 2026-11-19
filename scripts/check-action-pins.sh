#!/bin/sh
# Callers pin our Action by SHA and must get exactly that code, so it pins everything by SHA.
# Workflows hand secrets and GITHUB_TOKEN to third-party actions, so those pin by SHA too (ADR 0038).
set -u

usage="usage: check-action-pins.sh [--workflows DIR] [ACTION_FILE...]"
workflow_dir=""
if [ "${1:-}" = "--workflows" ]; then
  if [ $# -lt 2 ]; then
    echo "$usage" >&2
    exit 2
  fi
  workflow_dir=$2
  shift 2
fi

action_files="$*"
if [ -z "$action_files" ] && [ -z "$workflow_dir" ]; then
  action_files="action/action.yml"
fi

# A grep error on a missing input would otherwise read as "nothing unpinned".
for action_file in $action_files; do
  if [ ! -f "$action_file" ]; then
    echo "check-action-pins.sh: no such action file: $action_file" >&2
    echo "$usage" >&2
    exit 2
  fi
done
if [ -n "$workflow_dir" ] && [ ! -d "$workflow_dir" ]; then
  echo "check-action-pins.sh: no such workflow directory: $workflow_dir" >&2
  exit 2
fi

uses_pattern='^[[:space:]]*(-[[:space:]]+)?uses:[[:space:]]+[^.[:space:]]'
sha_pin_pattern='@[0-9a-f]{40}([[:space:]]|$)'

unpinned_in_actions=""
if [ -n "$action_files" ]; then
  unpinned_in_actions=$(grep -HnE "$uses_pattern" $action_files | grep -vE "$sha_pin_pattern")
fi

unpinned_in_workflows=""
if [ -n "$workflow_dir" ]; then
  unpinned_in_workflows=$(grep -rHnE "$uses_pattern" --include='*.yml' --include='*.yaml' "$workflow_dir" |
    grep -vE 'uses:[[:space:]]+actions/' | grep -vE "$sha_pin_pattern")
fi

if [ -n "$unpinned_in_actions$unpinned_in_workflows" ]; then
  echo "actions must be pinned to a full commit SHA (workflows may use a tag for actions/*):"
  [ -n "$unpinned_in_actions" ] && echo "$unpinned_in_actions"
  [ -n "$unpinned_in_workflows" ] && echo "$unpinned_in_workflows"
  exit 1
fi
echo "all actions pinned"
