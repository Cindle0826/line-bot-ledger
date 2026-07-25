resource "google_service_account" "webhook" {
  project      = var.project_id
  account_id   = "ledger-webhook-sa"
  display_name = "line-bot-ledger webhook service"
}

resource "google_service_account" "summary" {
  project      = var.project_id
  account_id   = "ledger-summary-sa"
  display_name = "line-bot-ledger summary service"
}

resource "google_service_account" "scheduler_invoker" {
  project      = var.project_id
  account_id   = "ledger-scheduler-invoker"
  display_name = "Invokes services/summary from Cloud Scheduler"
}

resource "google_service_account" "liff" {
  project      = var.project_id
  account_id   = "ledger-liff-sa"
  display_name = "line-bot-ledger liff service"
}

# Firestore access for all three services.
resource "google_project_iam_member" "webhook_datastore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.webhook.email}"
}

resource "google_project_iam_member" "summary_datastore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.summary.email}"
}

resource "google_project_iam_member" "liff_datastore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.liff.email}"
}

# webhook needs both secrets (verifies signature + calls Reply API).
resource "google_secret_manager_secret_iam_member" "webhook_channel_secret" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.line_channel_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.webhook.email}"
}

resource "google_secret_manager_secret_iam_member" "webhook_channel_token" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.line_channel_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.webhook.email}"
}

# summary only pushes messages, no signature to verify, so it only needs the
# channel token.
resource "google_secret_manager_secret_iam_member" "summary_channel_token" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.line_channel_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.summary.email}"
}

# liff doesn't call the Messaging API itself, but internal/base.New always
# initializes the bot client, so LINE_CHANNEL_TOKEN is required at startup
# regardless.
resource "google_secret_manager_secret_iam_member" "liff_channel_token" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.line_channel_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.liff.email}"
}
