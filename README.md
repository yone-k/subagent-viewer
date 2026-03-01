# cc-subagent-viewer

Claude Code のサブエージェントセッションをリアルタイムで監視するターミナル UI (TUI) ツール。

タスクの進捗状況、エージェントの会話内容、デバッグログ、セッション統計をひとつの画面で確認できます。

## インストール

### npm (推奨)

```bash
npm install -g cc-subagent-viewer
```

または `npx` でワンショット実行:

```bash
npx cc-subagent-viewer
```

### Go

Go 1.25 以上が必要です。

```bash
go install github.com/yone-k/cc-subagent-viewer@latest
```

または、リポジトリをクローンしてビルド:

```bash
git clone https://github.com/yone-k/cc-subagent-viewer.git
cd cc-subagent-viewer
go build -o cc-subagent-viewer .
```

## 使い方

### セッション選択モード

引数なしで起動すると、`~/.claude/history.jsonl` からセッション一覧を表示します。

```bash
cc-subagent-viewer
```

各セッションにはプロジェクトパス、相対時刻、最初のユーザー入力、利用可能なデータの種類 (`[Tasks|Logs]`) が表示されます。fuzzy フィルタリングで絞り込みが可能です。

### 直接指定モード

セッション UUID を引数に渡すと、セレクターをスキップして直接ビューアーを開きます。

```bash
cc-subagent-viewer 7ba50137-65c8-4349-b420-cdce14c38d2a
```

## タブ

### Tasks

サブエージェントのタスク一覧をリアルタイム表示します。

- 進捗バーで完了状況を可視化
- ステータスアイコン: `✓` 完了 / `●` 進行中 / `○` 保留 / `✗` ブロック中
- `Enter` でタスク詳細の展開/折畳み
- タスクファイルの変更を fsnotify で即時検知

### Agents

サブエージェントの一覧と会話内容を表示します。

- エージェントのステータス (`●` 実行中 / `✓` 完了)、タイプ、プロンプトを一覧表示
- `Enter` で選択したエージェントの会話ビューに切替
- 会話ビューではブロックタイプ (Text / Tool / Result / Thinking) でフィルタ可能
- `projects/{encoded}/{UUID}/subagents/agent-*.jsonl` を 1秒ポーリングで追跡

### Logs

デバッグログ (`~/.claude/debug/{UUID}.txt`) を 500ms ポーリングで追跡します。

- 7種類のログレベルを `Shift+←/→` で選択し `Enter` でフィルタトグル
- `/` でインライン検索（リアルタイム絞り込み）
- 最下部にいるとき自動スクロール
- 最大 10,000 エントリをリングバッファで保持

### Stats

`~/.claude.json` からセッション統計を表示します。

- コスト、所要時間、入出力トークン数
- モデル別の使用量内訳
- 最新セッション以外を表示中の場合は警告を表示（`.claude.json` は最新セッションの統計のみ保持するため）

## キーバインド

### グローバル

| キー | 動作 |
|------|------|
| `1` `2` `3` `4` | タブ切替 (Tasks / Agents / Logs / Stats) |
| `Tab` `→` / `Shift+Tab` `←` | 次 / 前のタブ |
| `q` / `Ctrl+C` | 終了 |

### Tasks

| キー | 動作 |
|------|------|
| `↑` / `k` | 上へ移動 |
| `↓` / `j` | 下へ移動 |
| `Enter` | 詳細展開 / 折畳み |

### Agents

| キー | 動作 |
|------|------|
| `↑` / `k` | 上へ移動 |
| `↓` / `j` | 下へ移動 |
| `Enter` | 会話ビューを開く |
| `Esc` | 一覧に戻る |

### Agents — 会話ビュー

| キー | 動作 |
|------|------|
| `Shift+←` / `Shift+→` | フィルタカーソル移動 (Text / Tool / Result / Thinking) |
| `Enter` | フィルタ切替 |
| `↑` `↓` / `k` `j` | スクロール |
| `Esc` | 一覧に戻る |

### Logs

| キー | 動作 |
|------|------|
| `Shift+←` / `Shift+→` | フィルタカーソル移動 (Debug / Error / Warn / MCP / Startup / Meta / Attach) |
| `Enter` | フィルタ切替 |
| `/` | 検索モード |
| `Esc` | 検索終了 |
| `↑` `↓` / `k` `j` | スクロール |

## データソース

すべて `~/.claude/` 配下のファイルを読み取り専用で参照します。

| パス | 内容 |
|------|------|
| `history.jsonl` | セッション履歴 (JSONL) |
| `tasks/{UUID}/*.json` | タスク状態。`.lock` はセッションがアクティブ中であることを示す |
| `debug/{UUID}.txt` | タイムスタンプ付きデバッグログ |
| `projects/{encoded}/{UUID}/subagents/agent-*.jsonl` | サブエージェント会話ログ |
| `projects/{encoded}/{UUID}.jsonl` | 親会話 (エージェント説明の補完に使用) |
| `~/.claude.json` | プロジェクト別統計 (最新セッションのみ) |

## 技術スタック

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — Elm アーキテクチャ TUI フレームワーク
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — ターミナルスタイリング
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI コンポーネント (list, textinput, key)
- [fsnotify](https://github.com/fsnotify/fsnotify) — クロスプラットフォームファイル監視

## ライセンス

MIT
