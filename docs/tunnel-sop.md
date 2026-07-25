# Cloudflare Tunnel SOP

本機測試三個服務用：`webhook`（8080）、`liff`（8081）、`summary`（8082）。`summary` 的 `/run` 在正式環境靠 Cloud Run IAM 只讓 Cloud Scheduler 呼叫，**本機/tunnel 沒有這層保護，`/run` 沒有任何驗證**——測完記得關掉 tunnel，別長時間曝露在公開網址上。

## 一次性設定

安裝 cloudflared

```bash
brew install cloudflared
```

登入並授權 domain

```bash
cloudflared tunnel login
```

建立 tunnel

```bash
cloudflared tunnel create line-bot-ledger-webhook
```

建立三條 DNS route

```bash
cloudflared tunnel route dns line-bot-ledger-webhook webhook-dev.你的domain.com
cloudflared tunnel route dns line-bot-ledger-webhook liff-dev.你的domain.com
cloudflared tunnel route dns line-bot-ledger-webhook summary-dev.你的domain.com
```

寫 `~/.cloudflared/config.yml`

```yaml
tunnel: line-bot-ledger-webhook
credentials-file: /Users/yihao.chen/.cloudflared/<TUNNEL_ID>.json
ingress:
  - hostname: webhook-dev.你的domain.com
    service: http://localhost:8080
  - hostname: liff-dev.你的domain.com
    service: http://localhost:8081
  - hostname: summary-dev.你的domain.com
    service: http://localhost:8082
  - service: http_status:404
```

## 每次測試

登入 GCP ADC（Firestore 存取用）

```bash
gcloud auth application-default login
```

啟動 tunnel（改過 `config.yml` 要重跑這行才會生效）

```bash
make tunnel
```

啟動本機服務（依需要挑）

```bash
make run-webhook
make run-liff
make run-summary
```

Webhook 設定頁

```
https://developers.line.biz/console/channel/2010837202/messaging-api
```

Webhook URL 填這個，存檔後按 Verify

```
https://webhook-dev.你的domain.com/callback
```

liff 設定見 [liff-sop.md](liff-sop.md)。

手動觸發一次 summary（模擬 Cloud Scheduler）

```bash
curl -s -X POST https://summary-dev.你的domain.com/run
```

## 快速檢查

本機健康檢查

```bash
curl -s http://localhost:8080/status
curl -s http://localhost:8081/status
curl -s http://localhost:8082/status
```

外部健康檢查（經過 tunnel）

```bash
curl -s https://webhook-dev.你的domain.com/status
curl -s https://liff-dev.你的domain.com/status
curl -s https://summary-dev.你的domain.com/status
```

Firestore 資源還沒 apply 之前，webhook 回「記帳失敗，請稍後再試」屬正常。

正式環境部署見 [deployment-sop.md](deployment-sop.md)。
