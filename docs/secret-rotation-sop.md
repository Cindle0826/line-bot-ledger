# 密鑰輪替 SOP

適用對象:`line-channel-secret`、`line-channel-token`(見 [env-vars.md](env-vars.md))。什麼情況要做這件事——密鑰外流、LINE 後台重新簽發、定期輪替。

## 關鍵前提:Cloud Run 不會自動套用新版本

`cloud_run.tf` 裡的 `secret_key_ref { version = "latest" }` 只在**部署新 revision 的當下**解析一次,不是「隨時去 Secret Manager 抓最新的」。單純加一個新版本,正在跑的 revision 完全不知道、繼續用舊值——一定要強制生出新 revision,新值才會生效。

## 步驟

1. 在 LINE Developers Console 重新簽發需要輪替的值(Channel secret 用 Messaging API 分頁的重新簽發,Channel access token 用 Reissue 按鈕),記下新值
2. 補一個新的 Secret Manager 版本:
   ```bash
   echo -n "<新的值>" | gcloud secrets versions add line-channel-token --project=line-bot-503410 --data-file=-
   ```
   （`line-channel-secret` 同理,換 secret 名稱）
3. **強制 Cloud Run 產生新 revision**,讓它重新解析 `latest`:
   ```bash
   gcloud run services update ledger-webhook --project=line-bot-503410 --region=asia-east1 --update-labels=rotated-at=$(date +%s)
   ```
   對 `ledger-summary`、`ledger-liff` 也各做一次(兩個都吃 `LINE_CHANNEL_TOKEN`)。加一個沒意義的 label 只是為了觸發新 revision,不影響任何行為
4. 更新本機 `.env` 裡對應的值(本機開發用,跟 Secret Manager 是分開的兩份副本,不會自動同步)
5. 確認新 revision 正常:
   ```bash
   curl -s https://<服務網址>/status
   ```
6. （可選)舊版本不用手動刪除——Secret Manager 免費額度是 6 個 active version,超過再清

## 什麼時候不需要做這件事

- `LINE_LOGIN_CHANNEL_ID`、`LIFF_ID` 都不是要保密的密鑰,值變了直接改 `terraform.tfvars`/`variables.tf` 重新 `apply` 就好,不用走這套流程
