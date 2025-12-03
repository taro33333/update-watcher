# Update Watcher 🔍

技術アップデート情報を自動で監視し、Slackに通知するツールです。

## 📋 機能

このツールは以下の情報源から**過去25時間の更新**を自動でチェックし、Slackに通知します：

### 通常情報

- 🔥 **GCP Release Notes** - Google Cloud Platform の最新リリース情報
- 🦫 **Go Releases** - Go言語の新しいリリース情報
- 🏗️ **Terraform Releases** - Terraformの新しいリリース情報

### セキュリティ情報（専用チャンネル対応 🔐）

- ☁️ **AWS Security Bulletins** - AWSのセキュリティ脆弱性情報
- 🔶 **Cloudflare Security Blog** - Cloudflareのセキュリティブログ記事
- 🔷 **GCP Security Bulletins** - Google Cloudのセキュリティ脆弱性情報
- 🐧 **Debian Security Advisories** - Debianのセキュリティ脆弱性情報
- 🛡️ **NVD CVE Database** - 国立脆弱性データベース（NVD）のCVE情報
- 🔐 **GitHub Security Advisories** - GitHub上のセキュリティ脆弱性情報

### 標準モード

`--json`を指定しなければ標準モードとしてSlackに通知が飛びます。

### JSON出力モード

```bash
# 標準出力にJSON形式で出力（Slack送信なし）
go run main.go --json
```

出力形式（JSON Lines）:

```json
{"url":"https://go.dev/blog/go1.21","project":"","title":"Go 1.21 Release Notes","summary":"..."}
{"url":"https://cloud.google.com/...","project":"","title":"GCP Update","summary":"..."}
```

### smart-digest との連携

[smart-digest](https://github.com/taro33333/smart-digest) と組み合わせて、AI による重要度判定と日本語要約を追加できます：

```bash
# update-watcher の出力を smart-digest にパイプ
go run main.go --json | smart-digest --threshold 75

# 結果を Slack に投稿
go run main.go --json | smart-digest | slack-post
```
