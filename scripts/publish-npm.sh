#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# publish-npm.sh
#
# GoReleaser のビルド成果物を npm パッケージに組み込んで公開するスクリプト。
# CIから呼び出されることを想定。
#
# Usage:
#   ./scripts/publish-npm.sh v0.1.0 [--dry-run]
# ---------------------------------------------------------------------------

# ---- プロジェクトルート特定 ------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ---- 引数パース ----------------------------------------------------------
VERSION=""
DRY_RUN=false

for arg in "$@"; do
  case "$arg" in
    --dry-run)
      DRY_RUN=true
      ;;
    *)
      if [[ -z "$VERSION" ]]; then
        VERSION="$arg"
      fi
      ;;
  esac
done

if [[ -z "$VERSION" ]]; then
  echo "Error: version argument is required (e.g. v0.1.0)" >&2
  exit 1
fi

# v プレフィックスを除去
VERSION="${VERSION#v}"

echo "Publishing version: ${VERSION}"
echo "Dry run: ${DRY_RUN}"

# ---- 環境変数チェック ----------------------------------------------------
if [[ "$DRY_RUN" == "false" && -z "${NODE_AUTH_TOKEN:-}" ]]; then
  echo "Error: NODE_AUTH_TOKEN is not set" >&2
  exit 1
fi

# ---- プラットフォームマッピング （bash 3.x 互換） --------------------------
# "npmディレクトリ名:GoReleaser distディレクトリ名" のペア
PLATFORMS="
darwin-arm64:cc-subagent-viewer_darwin_arm64_v8.0
darwin-x64:cc-subagent-viewer_darwin_amd64_v1
linux-x64:cc-subagent-viewer_linux_amd64_v1
linux-arm64:cc-subagent-viewer_linux_arm64_v8.0
"

# ---- 一時ディレクトリ ----------------------------------------------------
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

echo "Working directory: ${WORK_DIR}"

# ---- プラットフォームパッケージの準備 --------------------------------------
for entry in $PLATFORMS; do
  platform="${entry%%:*}"
  dist_dir="${entry##*:}"
  pkg_name="cc-subagent-viewer-${platform}"
  pkg_dir="${WORK_DIR}/${platform}"

  echo ""
  echo "=== Preparing ${pkg_name} ==="

  # テンプレートディレクトリをコピー
  cp -r "${PROJECT_ROOT}/npm/${platform}" "$pkg_dir"

  # package.json の version を更新
  jq --arg v "$VERSION" '.version = $v' "${pkg_dir}/package.json" > "${pkg_dir}/package.json.tmp"
  mv "${pkg_dir}/package.json.tmp" "${pkg_dir}/package.json"

  # バイナリをコピー
  mkdir -p "${pkg_dir}/bin"
  cp "${PROJECT_ROOT}/dist/${dist_dir}/cc-subagent-viewer" "${pkg_dir}/bin/cc-subagent-viewer"
  chmod +x "${pkg_dir}/bin/cc-subagent-viewer"

  echo "  Binary: dist/${dist_dir}/cc-subagent-viewer -> bin/cc-subagent-viewer"
done

# ---- メインパッケージの準備 ------------------------------------------------
echo ""
echo "=== Preparing cc-subagent-viewer (main) ==="

MAIN_PKG_DIR="${WORK_DIR}/cc-subagent-viewer"
cp -r "${PROJECT_ROOT}/npm/cc-subagent-viewer" "$MAIN_PKG_DIR"

# package.json の version を更新
jq --arg v "$VERSION" '.version = $v' "${MAIN_PKG_DIR}/package.json" > "${MAIN_PKG_DIR}/package.json.tmp"
mv "${MAIN_PKG_DIR}/package.json.tmp" "${MAIN_PKG_DIR}/package.json"

# optionalDependencies の各バージョンも更新
jq --arg v "$VERSION" '.optionalDependencies |= with_entries(.value = $v)' "${MAIN_PKG_DIR}/package.json" > "${MAIN_PKG_DIR}/package.json.tmp"
mv "${MAIN_PKG_DIR}/package.json.tmp" "${MAIN_PKG_DIR}/package.json"

echo "  version and optionalDependencies updated to ${VERSION}"

# ---- 公開 ----------------------------------------------------------------
if [[ "$DRY_RUN" == "true" ]]; then
  echo ""
  echo "=== Dry run: showing package.json files ==="
  for entry in $PLATFORMS; do
    platform="${entry%%:*}"
    pkg_dir="${WORK_DIR}/${platform}"
    echo ""
    echo "--- ${platform}/package.json ---"
    cat "${pkg_dir}/package.json"
  done
  echo ""
  echo "--- cc-subagent-viewer/package.json ---"
  cat "${MAIN_PKG_DIR}/package.json"
  echo ""
  echo "Dry run complete. No packages were published."
  exit 0
fi

# プラットフォームパッケージを先に公開
for entry in $PLATFORMS; do
  platform="${entry%%:*}"
  pkg_dir="${WORK_DIR}/${platform}"
  pkg_name="cc-subagent-viewer-${platform}"

  echo ""
  echo "=== Publishing ${pkg_name}@${VERSION} ==="
  cd "$pkg_dir"
  if ! output=$(npm publish --access public 2>&1); then
    if echo "$output" | grep -q "You cannot publish over the previously published versions"; then
      echo "  Already published, skipping."
    else
      echo "$output" >&2
      exit 1
    fi
  else
    echo "  Published successfully."
  fi
done

# メインパッケージを最後に公開
echo ""
echo "=== Publishing cc-subagent-viewer@${VERSION} ==="
cd "$MAIN_PKG_DIR"
if ! output=$(npm publish --access public 2>&1); then
  if echo "$output" | grep -q "You cannot publish over the previously published versions"; then
    echo "  Already published, skipping."
  else
    echo "$output" >&2
    exit 1
  fi
else
  echo "  Published successfully."
fi

echo ""
echo "All packages published successfully!"
