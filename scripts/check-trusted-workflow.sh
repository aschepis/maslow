#!/bin/sh
# A pull_request_target workflow runs the change's code under main's definition, so that code must
# never get write scope, secrets, a token left on disk or a cache another run restores (ADR 0040).
set -u

usage="usage: check-trusted-workflow.sh WORKFLOW_FILE..."
if [ $# -eq 0 ]; then
  echo "$usage" >&2
  exit 2
fi
for workflow in "$@"; do
  if [ ! -f "$workflow" ]; then
    echo "check-trusted-workflow.sh: no such workflow file: $workflow" >&2
    echo "$usage" >&2
    exit 2
  fi
done

guard='github.event.pull_request.head.repo.full_name == github.repository'

# Reads the block-style YAML GitHub workflows use; flow mappings are not parsed.
program='
function trim(s) { sub(/^[ \t]+/, "", s); sub(/[ \t]+$/, "", s); return s }
function unquote(s) {
  if (length(s) >= 2 && (substr(s, 1, 1) == "\"" || substr(s, 1, 1) == sq) && substr(s, length(s), 1) == substr(s, 1, 1))
    return substr(s, 2, length(s) - 2)
  return s
}
function report(msg) { print FILENAME ": " msg; bad = 1 }
function report_line(msg) { print FILENAME ":" FNR ": " msg; bad = 1 }
function end_step() {
  if (!in_step) return
  if (step_uses ~ /^actions\/checkout@/ && step_persist != "false")
    report("job " job ": actions/checkout must set persist-credentials: false")
  if (step_uses ~ /^actions\/setup-go@/ && step_cache != "false")
    report("job " job ": actions/setup-go must set cache: false")
  if (step_uses ~ /^actions\/cache(\/[^@]*)?@/)
    report("job " job ": actions/cache is not allowed")
  in_step = 0; step_uses = ""; step_persist = ""; step_cache = ""
}
function end_job() {
  end_step()
  if (job != "" && !guarded)
    report("job " job " must run only when head.repo.full_name == github.repository")
  job = ""; guarded = 0; in_if = 0; in_steps = 0
}
BEGIN { job_indent = -1; perms = "<missing>" }
{
  line = $0
  if (line ~ /^[ \t]*(#|$)/) next
  sub(/[ \t]+#.*$/, "", line)
  match(line, /^ */)
  indent = RLENGTH
  text = substr(line, indent + 1)

  t = text
  sub(/^-[ \t]+/, "", t)
  key = ""; value = ""
  if (match(t, /^"?[A-Za-z0-9_-]+"?:/)) {
    rest = substr(t, RLENGTH + 1)
    if (rest == "" || rest ~ /^[ \t]/) {
      key = substr(t, 1, RLENGTH - 1)
      gsub(/"/, "", key)
      value = unquote(trim(rest))
    }
  }

  if (line ~ /secrets\.|secrets\[|github\.token|secrets:[ \t]*inherit/)
    report_line("references secrets or github.token")
  if (key != "" && (value == "write" || value == "write-all"))
    report_line("grants a write permission")

  if (indent == 0) {
    end_job()
    top = key
    if (key == "on") {
      triggers = value
      gsub(/\[/, "", triggers); gsub(/\]/, "", triggers); gsub(/ /, "", triggers)
      on_indent = -1
    }
    if (key == "permissions") perms = value
    next
  }

  if (top == "on") {
    if (on_indent < 0) on_indent = indent
    if (indent == on_indent && key != "") triggers = (triggers == "" ? key : triggers "," key)
    next
  }

  if (top != "jobs") next
  if (job_indent < 0) job_indent = indent
  if (indent == job_indent) {
    end_job()
    job = key
    child_indent = -1
    next
  }
  if (child_indent < 0) child_indent = indent
  if (indent == child_indent) {
    end_step()
    in_steps = (key == "steps")
    in_if = (key == "if")
    step_indent = -1
    if (in_if && index(value, guard)) guarded = 1
    next
  }
  if (in_if && index(text, guard)) guarded = 1
  if (!in_steps) next

  if (text ~ /^-([ \t]|$)/ && (step_indent < 0 || indent == step_indent)) {
    end_step()
    step_indent = indent
    in_step = 1
  }
  if (in_step) {
    if (key == "uses") step_uses = value
    if (key == "persist-credentials") step_persist = value
    if (key == "cache") step_cache = value
  }
}
END {
  end_job()
  if (triggers != "pull_request_target")
    report("the only trigger must be pull_request_target (found: " triggers ")")
  if (perms != "{}")
    report("top-level permissions must be {}")
  exit bad
}
'

status=0
for workflow in "$@"; do
  awk -v sq="'" -v guard="$guard" "$program" "$workflow"
  code=$?
  if [ "$code" -gt 1 ]; then
    exit 2
  fi
  if [ "$code" -eq 1 ]; then
    status=1
  fi
done

if [ "$status" -ne 0 ]; then
  exit 1
fi
echo "trusted workflow invariants hold"
