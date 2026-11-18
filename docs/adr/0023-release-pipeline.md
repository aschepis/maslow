# ADR 0023: Release Pipeline

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

A pipeline that gates on Maslow must pin a version and know that the binary it runs is the one that was published. There were no released binaries, and the version in `maslow version` came from ldflags nothing set.

## Decision

Pushing a `v*` tag runs tests, lint and `maslow verify --profile full`, then `scripts/build-release.sh` builds `maslow_<version>_<os>_<arch>.tar.gz` for linux and darwin on amd64 and arm64 plus a `checksums.txt`, and `gh release create` publishes them.

## Consequences

- The version is stamped into `internal/buildinfo` and used by both `maslow version` and the evidence's `maslow_version`. A `go install ...@vX.Y.Z` binary reports the same string from its module version, and a leading `v` is always dropped.
- Release logic lives in a script rather than a third-party release tool, so CI runs the same build as a smoke test on every push and it can be run locally.
- Windows is not built. Adding it means a `.zip` and an `.exe` name in the script.
