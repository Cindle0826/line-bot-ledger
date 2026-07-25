locals {
  apis = [
    "run.googleapis.com",
    "firestore.googleapis.com",
    "secretmanager.googleapis.com",
    "cloudscheduler.googleapis.com",
    "artifactregistry.googleapis.com",
    "iam.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    # Workload Identity Federation (github_actions.tf): iamcredentials for
    # short-lived token generation, sts for the OIDC token exchange. No
    # cloudbuild.googleapis.com anymore — GitHub Actions builds and pushes
    # images directly now, Cloud Build is unused.
    "iamcredentials.googleapis.com",
    "sts.googleapis.com",
    "billingbudgets.googleapis.com",
  ]
}

resource "google_project_service" "apis" {
  for_each           = toset(local.apis)
  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}
