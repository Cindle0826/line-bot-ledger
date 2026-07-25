# line-bot-ledger

個人記帳 LINE Bot——傳文字訊息記一筆、LIFF 表單/歷史/訂閱設定、每日/每週/每月分類總結推播。三個獨立 Cloud Run 服務（`webhook`/`summary`/`liff`），Terraform 管理全部 GCP 基礎設施，GitHub Actions 打版號 tag 部署。

## 加好友

<img src="docs/line-qrcode.png" alt="LINE Bot QR Code" width="200" />

或直接開 [line.me/R/ti/p/@998fafjm](https://line.me/R/ti/p/@998fafjm)。

## 架構

完整架構圖、資料流、CI/CD 流程見 [docs/architecture.html](docs/architecture.html)。
