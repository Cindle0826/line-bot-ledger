resource "google_cloud_scheduler_job" "summary_daily" {
  project   = var.project_id
  region    = var.region
  name      = "ledger-summary-daily"
  schedule  = "0 22 * * *"
  time_zone = "Asia/Taipei"

  http_target {
    uri         = "${google_cloud_run_v2_service.summary.uri}/run"
    http_method = "POST"

    oidc_token {
      service_account_email = google_service_account.scheduler_invoker.email
      audience              = google_cloud_run_v2_service.summary.uri
    }
  }

  depends_on = [
    google_project_service.apis,
    google_cloud_run_v2_service_iam_member.summary_scheduler_invoker,
  ]
}
