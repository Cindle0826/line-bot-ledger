variable "project_id" {
  description = "GCP project ID"
  type        = string
  default     = "line-bot-503410"
}

variable "region" {
  description = "GCP region for all resources"
  type        = string
  default     = "asia-east1"
}

variable "billing_account" {
  description = "Billing account ID for the budget alert. Format: XXXXXX-XXXXXX-XXXXXX. Find with: gcloud billing accounts list"
  type        = string
}

variable "budget_amount" {
  description = "Monthly budget alert threshold, in the billing account's own currency (TWD for this account)"
  type        = number
  default     = 1
}

variable "webhook_image" {
  description = "Container image for services/webhook. Defaults to a public placeholder until .github/workflows/webhook.yml pushes the real image."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "summary_image" {
  description = "Container image for services/summary. Defaults to a public placeholder until .github/workflows/summary.yml pushes the real image."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "liff_image" {
  description = "Container image for services/liff. Defaults to a public placeholder until .github/workflows/liff.yml pushes the real image."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "github_repository" {
  description = "GitHub \"owner/repo\" trusted by Workload Identity Federation (github_actions.tf) to deploy from. Only workflow runs from this exact repo can impersonate the deployer service account. Case-sensitive — GitHub's OIDC token's assertion.repository claim preserves the account's actual casing (\"Cindle0826\", not \"cindle0826\"), and the attribute_condition comparison is a case-sensitive string match."
  type        = string
  default     = "Cindle0826/line-bot-ledger"
}

variable "line_login_channel_id" {
  description = "Channel ID of the LINE Login channel (\"Yihao 記帳 LIFF\") services/liff verifies ID tokens against. Not secret — it's the OAuth client_id, sent as a request parameter."
  type        = string
  default     = "2010843613"
}

variable "liff_id" {
  description = "LIFF app ID from the LINE Login channel's LIFF tab. Not secret — it ends up embedded in the public HTML the browser loads. Empty until the LIFF app is added in the console (see docs/liff-sop.md); services/liff won't start without it."
  type        = string
  default     = ""
}
