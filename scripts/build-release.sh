#!/usr/bin/env bash
# Builds release archives and a checksums file into dist/.
# Usage: scripts/build-release.sh <version>   e.g. scripts/build-release.sh v0.6.0
set -euo pipefail

version="${1:?usage: scripts/build-release.sh <version>}"
version="${version#v}"
commit="$(git rev-parse HEAD)"
pkg="github.com/aschepis/maslow/internal/buildinfo"
ldflags="-s -w -X ${pkg}.version=${version} -X ${pkg}.gitSHA=${commit}"

rm -rf dist
mkdir -p dist

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  os="${target%/*}"
  arch="${target#*/}"
  name="maslow_${version}_${os}_${arch}"
  stage="dist/${name}"
  mkdir -p "${stage}"
  CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" \
    go build -trimpath -ldflags "${ldflags}" -o "${stage}/maslow" ./cmd/maslow
  cp LICENSE README.md "${stage}/"
  tar -C dist -czf "dist/${name}.tar.gz" "${name}"
  rm -rf "${stage}"
done

(
  cd dist
  if command -v sha256sum >/dev/null; then
    sha256sum ./*.tar.gz | sed 's# \./# #' > checksums.txt
  else
    shasum -a 256 ./*.tar.gz | sed 's# \./# #' > checksums.txt
  fi
)

echo "built dist/ for ${version}:"
cat dist/checksums.txt
