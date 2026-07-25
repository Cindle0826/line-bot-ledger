# LIFF 記帳表單 SOP

`services/liff` 用 LIFF ID Token 驗證使用者，不是 Webhook 簽名、不是 GCP IAM。頁面自己寫（`services/liff/static/`），LINE 只存 LIFF ID 對應到 Endpoint URL。

## 建立 LINE Login channel

前提：provider 已存在。開 Console

```
https://developers.line.biz/console/
```

Create a new channel → LINE Login，App types 勾這個（LIFF 只認 Web app，不是 Mobile app）

```
Web app
```

## 新增 LIFF app

進 channel 的 LIFF 分頁 → Add，依序填：

Endpoint URL（本機開發）

```
https://liff-dev.你的domain.com/
```

Endpoint URL（正式環境，見 [deployment-sop.md](deployment-sop.md)）

```
https://<ledger-liff Cloud Run URL>/
```

Size

```
Compact
```

Scopes（兩個都要勾，缺 `openid` 會導致 `getIDToken()` 拿不到 token）

```
openid
profile
```

## 設定環境變數

拿到 LIFF ID 後填進 `.env`（三個服務共用的變數見 [env-vars.md](env-vars.md)）

```bash
LINE_LOGIN_CHANNEL_ID=<LINE Login channel 的 Channel ID>
LIFF_ID=<剛拿到的 LIFF ID>
```

## 測試

重啟服務

```bash
make run-liff
```

用手機開

```
https://liff.line.me/<LIFF_ID>
```

## 常見卡點

- 送出表單回 401：Scopes 沒勾 `openid`，回「新增 LIFF app」步驟補勾
- 頁面打不開：tunnel 沒重啟——改過 `~/.cloudflared/config.yml` 一定要重跑 `make tunnel`
- 服務啟動就失敗：`LIFF_ID` 或 `LINE_LOGIN_CHANNEL_ID` 沒填，`base.RequireEnv` 直接擋下來，不是 bug
