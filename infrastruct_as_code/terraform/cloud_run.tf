resource "google_cloud_run_v2_service" "webhook" {
  project             = var.project_id
  name                = "ledger-webhook"
  location            = var.region
  deletion_protection = false

  template {
    service_account = google_service_account.webhook.email

    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }

    containers {
      image = var.webhook_image

      env {
        name  = "GOOGLE_CLOUD_PROJECT"
        value = var.project_id
      }

      env {
        name = "LINE_CHANNEL_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.line_channel_secret.secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "LINE_CHANNEL_TOKEN"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.line_channel_token.secret_id
            version = "latest"
          }
        }
      }
    }
  }

  depends_on = [
    google_project_service.apis,
    google_secret_manager_secret_iam_member.webhook_channel_secret,
    google_secret_manager_secret_iam_member.webhook_channel_token,
  ]
}

# Public: LINE's platform must be able to reach /callback.
resource "google_cloud_run_v2_service_iam_member" "webhook_public" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.webhook.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_service" "summary" {
  project             = var.project_id
  name                = "ledger-summary"
  location            = var.region
  deletion_protection = false

  template {
    service_account = google_service_account.summary.email

    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }

    containers {
      image = var.summary_image

      env {
        name  = "GOOGLE_CLOUD_PROJECT"
        value = var.project_id
      }

      env {
        name = "LINE_CHANNEL_TOKEN"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.line_channel_token.secret_id
            version = "latest"
          }
        }
      }
    }
  }

  depends_on = [
    google_project_service.apis,
    google_secret_manager_secret_iam_member.summary_channel_token,
  ]
}

# Private: only the Cloud Scheduler invoker identity may call it.
resource "google_cloud_run_v2_service_iam_member" "summary_scheduler_invoker" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.summary.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.scheduler_invoker.email}"
}

resource "google_cloud_run_v2_service" "liff" {
  project             = var.project_id
  name                = "ledger-liff"
  location            = var.region
  deletion_protection = false

  template {
    service_account = google_service_account.liff.email

    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }

    containers {
      image = var.liff_image

      env {
        name  = "GOOGLE_CLOUD_PROJECT"
        value = var.project_id
      }

      env {
        name  = "LINE_LOGIN_CHANNEL_ID"
        value = var.line_login_channel_id
      }

      env {
        name  = "LIFF_ID"
        value = var.liff_id
      }

      env {
        name = "LINE_CHANNEL_TOKEN"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.line_channel_token.secret_id
            version = "latest"
          }
        }
      }
    }
  }

  depends_on = [
    google_project_service.apis,
    google_secret_manager_secret_iam_member.liff_channel_token,
  ]
}

# Public: the user's browser (via LINE's in-app browser) must be able to
# reach the form and POST /liff/entries. Auth is the LIFF ID token verified
# inside the app, not Cloud Run's own IAM layer — same trust model as
# webhook_public above.
resource "google_cloud_run_v2_service_iam_member" "liff_public" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.liff.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
