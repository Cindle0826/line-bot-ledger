# These create empty secret containers only. Actual values are added
# out-of-band so they never end up in .tf files, plan output, or state as
# a version's data — see infrastruct_as_code/terraform/README.md.

resource "google_secret_manager_secret" "line_channel_secret" {
  project   = var.project_id
  secret_id = "line-channel-secret"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret" "line_channel_token" {
  project   = var.project_id
  secret_id = "line-channel-token"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}
