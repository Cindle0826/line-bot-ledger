# Lets .github/workflows/webhook.yml and summary.yml authenticate to GCP
# without a long-lived JSON service account key: GitHub issues a short-lived
# OIDC token per workflow run, GCP exchanges it for a short-lived access
# token as google_service_account.github_actions_deployer below — but only
# for runs coming from var.github_repository (see attribute_condition on
# the provider). AWS analogy: this is the GCP equivalent of an IAM OIDC
# identity provider + assume-role trust policy for GitHub Actions.

resource "google_iam_workload_identity_pool" "github_actions" {
  project                   = var.project_id
  workload_identity_pool_id = "github-actions"
  display_name              = "GitHub Actions"

  depends_on = [google_project_service.apis]
}

resource "google_iam_workload_identity_pool_provider" "github_actions" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.github_actions.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub"

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
  }

  # Reject tokens from any repo other than this one.
  attribute_condition = "assertion.repository == \"${var.github_repository}\""

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account" "github_actions_deployer" {
  project      = var.project_id
  account_id   = "github-actions-deployer"
  display_name = "Deploys webhook/summary/liff from GitHub Actions"
}

# Lets workflow runs from var.github_repository impersonate the deployer SA.
resource "google_service_account_iam_member" "github_actions_wif" {
  service_account_id = google_service_account.github_actions_deployer.name
  role                = "roles/iam.workloadIdentityUser"
  member              = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github_actions.name}/attribute.repository/${var.github_repository}"
}

# Push access to the one Artifact Registry repo — not project-wide.
resource "google_artifact_registry_repository_iam_member" "github_actions_ar_writer" {
  project    = var.project_id
  location   = var.region
  repository = google_artifact_registry_repository.images.repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.github_actions_deployer.email}"
}

# Deploy access to each Cloud Run service individually — not project-wide,
# and deliberately not roles/run.admin, so it can't touch IAM policy
# (Terraform still owns webhook_public / summary_scheduler_invoker).
resource "google_cloud_run_v2_service_iam_member" "github_actions_deploy_webhook" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.webhook.name
  role     = "roles/run.developer"
  member   = "serviceAccount:${google_service_account.github_actions_deployer.email}"
}

resource "google_cloud_run_v2_service_iam_member" "github_actions_deploy_summary" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.summary.name
  role     = "roles/run.developer"
  member   = "serviceAccount:${google_service_account.github_actions_deployer.email}"
}

resource "google_cloud_run_v2_service_iam_member" "github_actions_deploy_liff" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.liff.name
  role     = "roles/run.developer"
  member   = "serviceAccount:${google_service_account.github_actions_deployer.email}"
}

# Deploying a revision requires the deployer to be allowed to "act as" the
# runtime service account, even when that account isn't changing.
resource "google_service_account_iam_member" "github_actions_act_as_webhook" {
  service_account_id = google_service_account.webhook.name
  role                = "roles/iam.serviceAccountUser"
  member              = "serviceAccount:${google_service_account.github_actions_deployer.email}"
}

resource "google_service_account_iam_member" "github_actions_act_as_summary" {
  service_account_id = google_service_account.summary.name
  role                = "roles/iam.serviceAccountUser"
  member              = "serviceAccount:${google_service_account.github_actions_deployer.email}"
}

resource "google_service_account_iam_member" "github_actions_act_as_liff" {
  service_account_id = google_service_account.liff.name
  role                = "roles/iam.serviceAccountUser"
  member              = "serviceAccount:${google_service_account.github_actions_deployer.email}"
}
