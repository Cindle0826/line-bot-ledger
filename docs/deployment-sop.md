# 部署流程 SOP

從「本機測試通過」到「Cloud Run 跑的是真的程式碼」要做的事。前提:`infrastruct_as_code/terraform/` 的資源已經 `apply` 過(Cloud Run 服務、Artifact Registry、Workload Identity Federation 都要先存在)。

## 一次性設定

1. 建立 GitHub repo,把這個本機 repo 推上去(`git remote add origin ...` + `git push -u origin main`)
2. 從 Terraform output 拿兩個值:
   ```bash
   cd infrastruct_as_code/terraform
   terraform output -raw github_actions_workload_identity_provider
   terraform output -raw github_actions_service_account_email
   ```
3. 到 GitHub repo → Settings → Secrets and variables → Actions,新增兩個 repository secret:
   - `GCP_WORKLOAD_IDENTITY_PROVIDER` = 上面第一個值
   - `GCP_SERVICE_ACCOUNT_EMAIL` = 上面第二個值

## 之後每次部署

1. 改 `services/<name>/` 或 `internal/` 底下的程式碼,commit、push 到 `main`(或開 PR)
2. **PR 階段**:`.github/workflows/<name>.yml` 只跑 build/test → lint → vulncheck,不會部署
3. **push 到 `main`**:上面三項過了之後,自動多跑 build image(`linux/amd64`)→ push Artifact Registry → `deploy-cloudrun`,把新 image 部署到對應的 Cloud Run 服務
4. Path filter 只認對應服務的目錄——只改 `services/webhook/**` 只會觸發 `webhook.yml`,不會動到 `summary`/`liff`;改 `internal/**` 三個 workflow 都會觸發,因為是共用套件

## 確認部署成功

```bash
gcloud run services describe ledger-webhook --project=line-bot-503410 --region=asia-east1 --format="value(status.latestReadyRevisionName,status.url)"
```

或直接看 GitHub Actions 的 workflow run 記錄,`deploy` job 綠燈就代表 `deploy-cloudrun` 這步跑完了。

## 把 LINE 端點從本機 tunnel 換成正式 Cloud Run URL

本機開發時,LINE 後台填的是 Cloudflare Tunnel 網址(`webhook-dev.growlabtech.com`、`liff-dev.growlabtech.com`)。第一次部署到 Cloud Run、確認服務正常後:

1. `terraform output webhook_url`(或直接 `gcloud run services describe ledger-webhook ...`)拿到正式網址
2. LINE Developers Console → Messaging API 分頁 → Webhook URL 改成 `<正式網址>/callback` → Verify → 確認 Use webhook 還是開著
3. LINE Login channel → LIFF 分頁 → 對應的 LIFF app → Endpoint URL 改成 `<ledger-liff 正式網址>/`
4. 確認好友歡迎訊息裡的 LIFF 連結(`https://liff.line.me/<LIFF_ID>`)不用改——那個是 LIFF ID 組出來的固定短網址,LINE 自己會轉址到你設定的 Endpoint URL,不受這次改動影響

## 常見卡點

- **Cloud Run 服務還沒 apply**:`deploy-cloudrun` 這步會直接失敗,因為服務不存在——先確認 `infrastruct_as_code/terraform` 的對應資源已經 apply
- **Secret Manager 是空容器**:即使 Cloud Run 服務建立成功,revision 也會卡住起不來(`secret_key_ref` 指到不存在的 version)——見下方密鑰輪替 SOP 的「第一次補值」步驟
- **repo secrets 沒設或設錯**:`deploy` job 在 `google-github-actions/auth` 這步就會失敗,回頭檢查兩個 GitHub repo secret 的值有沒有貼對
