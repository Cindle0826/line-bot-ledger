# 環境變數 / 密鑰總覽

三個服務用到的所有變數,誰要什麼、算不算密鑰、值放在哪裡。

## 各服務需要的變數

| 變數 | webhook | summary | liff | 算密鑰？ |
|---|---|---|---|---|
| `PORT` | ✓ | ✓ | ✓ | 否 |
| `GOOGLE_CLOUD_PROJECT` | ✓ | ✓ | ✓ | 否 |
| `LINE_CHANNEL_TOKEN` | ✓ | ✓ | ✓ | **是** |
| `LINE_CHANNEL_SECRET` | ✓ | | | **是** |
| `LINE_LOGIN_CHANNEL_ID` | | | ✓ | 否(OAuth client_id,設計上是公開的) |
| `LIFF_ID` | | | ✓ | 否(會直接嵌進公開 HTML) |

三個服務都透過 `internal/base.New(ctx)` 讀取共用變數,所以即使某服務用不到 `LINE_CHANNEL_TOKEN`(例如目前的 `summary`/`liff` 還沒真的呼叫 LINE Bot API),沒填一樣會啟動失敗——`base.New` 是無條件讀取的。

## 值放在哪裡

| 環境 | 存放位置 | 誰寫入 |
|---|---|---|
| 本機開發 | 根目錄 `.env`(gitignored) | 手動填,明文 |
| Cloud Run(密鑰類) | GCP Secret Manager(`line-channel-secret`、`line-channel-token`) | `gcloud secrets versions add`,不經過 Terraform |
| Cloud Run(非密鑰類) | Terraform 變數 → Cloud Run 環境變數,直接寫在 `google_cloud_run_v2_service` 的 `env` 區塊 | `terraform apply`,值來自 `terraform.tfvars` 或 `variables.tf` 的 default |

## 對照 Terraform 變數

| Terraform 變數 | 對應到哪個環境變數 | 定義在 |
|---|---|---|
| `var.line_login_channel_id` | `LINE_LOGIN_CHANNEL_ID`(liff) | `variables.tf`,default 已填真實值 |
| `var.liff_id` | `LIFF_ID`(liff) | `variables.tf`,default 空字串,要另外填進 `terraform.tfvars` |
| `var.webhook_image`/`summary_image`/`liff_image` | Cloud Run 的 container image | `variables.tf`,default 是 placeholder,CI 部署後才會覆蓋成真的 |

`LINE_CHANNEL_SECRET`/`LINE_CHANNEL_TOKEN` **沒有對應的 Terraform 變數**——這是刻意的,密鑰值不該出現在任何 `.tf`/`.tfvars`/state 裡,只用 `secret_key_ref` 指向 Secret Manager 的 secret ID,值另外用 `gcloud secrets versions add` 補。

**`summary` 推播對象不是環境變數**:誰要收到每日/每週/每月總結,存在 Firestore `users/{lineUserID}.subscriptions` 欄位裡,由 `services/liff` 的 `/settings` 頁面寫入,`summary` 每次 `/run` 用 `store.ListSubscribers` 查——沒有任何 Terraform 變數或 Cloud Run 環境變數控制這件事。

## 新增一個變數時

1. 只是設定值(不是密鑰):加進對應服務的 `internal/base`(如果三個服務都要用)或 `main.go`(單一服務專用)裡的 `base.RequireEnv(...)`,`variables.tf` 加一個 Terraform 變數,`cloud_run.tf` 的 `env` 區塊加一筆
2. 是密鑰:`secrets.tf` 加一個 `google_secret_manager_secret` 容器,`iam.tf` 給需要的服務 `roles/secretmanager.secretAccessor`,`cloud_run.tf` 用 `secret_key_ref` 接,**絕對不要**加 Terraform 變數或寫進任何 `.tf`/`.tfvars`
3. 兩種都要記得更新 `.env.example`(只列名稱,不填值)和這份文件的表格
