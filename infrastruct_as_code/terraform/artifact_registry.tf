resource "google_artifact_registry_repository" "images" {
  project       = var.project_id
  location      = var.region
  repository_id = "line-bot-ledger"
  format        = "DOCKER"

  # Keep the 5 most recent versions always; delete anything else once it's
  # over 30 days old. Artifact Registry's free tier is only 0.5GiB/month, so
  # without this, repeated `gcloud builds submit` runs will quietly grow past
  # it.
  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 5
    }
  }

  cleanup_policies {
    id     = "delete-old"
    action = "DELETE"
    condition {
      tag_state  = "ANY"
      older_than = "2592000s" # 30 days
    }
  }

  depends_on = [google_project_service.apis]
}
