"use strict";

const SUPPORTED_PLATFORMS = {
  "darwin-arm64": "cc-subagent-viewer-darwin-arm64",
  "darwin-x64": "cc-subagent-viewer-darwin-x64",
  "linux-x64": "cc-subagent-viewer-linux-x64",
  "linux-arm64": "cc-subagent-viewer-linux-arm64",
};

const platformKey = `${process.platform}-${process.arch}`;
const packageName = SUPPORTED_PLATFORMS[platformKey];

if (!packageName) {
  console.warn(
    `警告: cc-subagent-viewer はこのプラットフォーム (${process.platform}-${process.arch}) をサポートしていません。\n` +
    `サポート対象: ${Object.keys(SUPPORTED_PLATFORMS).join(", ")}`
  );
  // Don't fail the install - exit cleanly
  process.exit(0);
}

try {
  require.resolve(`${packageName}/package.json`);
} catch (e) {
  console.warn(
    `警告: プラットフォームパッケージ ${packageName} がインストールされていません。\n` +
    `cc-subagent-viewer が正常に動作しない可能性があります。\n` +
    `npm install を --no-optional なしで再実行してください。`
  );
}

// Always exit cleanly to not block the install
process.exit(0);
