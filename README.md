# Update Watcher 🔍

技術アップデート情報を自動で監視し、Slackに通知するツールです。

## 📋 機能

このツールは以下の情報源から**過去25時間の更新**を自動でチェックし、Slackに通知します：

### 通常情報
- 🔥 **GCP Release Notes** - Google Cloud Platform の最新リリース情報
- 🦫 **Go Releases** - Go言語の新しいリリース情報
- 🏗️ **Terraform Releases** - Terraformの新しいリリース情報

### セキュリティ情報（専用チャンネル対応 🔐）
- 🐧 **Debian Security Advisories** - Debianのセキュリティ脆弱性情報
- 🔐 **GitHub Security Advisories** - GitHub上のセキュリティ脆弱性情報

> セキュリティ情報は、別のSlackチャンネルに通知することができます。

## ⚙️ 環境変数

| 変数名 | 必須 | 説明 |
|--------|------|------|
| `SLACK_WEBHOOK_URL` | ✅ 必須 | 通常情報の通知先Slack Webhook URL |
| `SLACK_SECURITY_WEBHOOK_URL` | ⭕ オプション | セキュリティ情報専用のSlack Webhook URL<br>未設定の場合は `SLACK_WEBHOOK_URL` を使用 |
| `GITHUB_TOKEN` | ⭕ オプション | GitHub API のレート制限回避用トークン<br>未設定でも動作しますが、設定を推奨 |

### 使用例

```bash
# 通常情報とセキュリティ情報を同じチャンネルに送信
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
go run main.go

# セキュリティ情報を専用チャンネルに送信
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
export SLACK_SECURITY_WEBHOOK_URL="https://hooks.slack.com/services/SECURITY/WEBHOOK/URL"
go run main.go
```

## 🏗️ プロジェクト構成

堅牢性、パフォーマンス、可読性を兼ね備えた「Goらしい」設計：

```
update-watcher/
├── main.go                    # エントリーポイント (120行)
├── internal/
│   ├── checker/              # チェッカーインターフェース・型定義
│   │   ├── checker.go
│   │   └── types.go
│   ├── client/               # HTTPクライアント
│   │   └── http.go
│   ├── config/               # 設定・定数
│   │   └── config.go
│   ├── notifier/             # Slack通知
│   │   └── slack.go
│   ├── sources/              # 各情報源の実装
│   │   ├── gcp.go
│   │   ├── golang.go
│   │   ├── terraform.go
│   │   ├── debian.go
│   │   └── github.go
│   └── util/                 # ユーティリティ
│       ├── date.go
│       └── text.go
└── go.mod
```

### 設計の特徴

- ✅ **責任の分離**: 各パッケージが明確な責任を持つ
- ✅ **依存性注入**: テスタビリティの向上
- ✅ **インターフェース駆動**: 拡張性の確保
- ✅ **共通HTTPクライアント**: パフォーマンス最適化
- ✅ **Context対応**: タイムアウト・キャンセル制御
- ✅ **セキュリティ情報の分離**: 専用チャンネルでの通知に対応
