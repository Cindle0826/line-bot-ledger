output "webhook_url" {
  description = "Public URL for services/webhook. Set this + /callback as the LINE Webhook URL."
  value       = google_cloud_run_v2_service.webhook.uri
}

output "summary_url" {
  description = "Private URL for services/summary. Only the scheduler invoker service account can call it."
  value       = google_cloud_run_v2_service.summary.uri
}

output "artifact_registry_repo" {
  description = "Base Artifact Registry path (matches the IMAGE env var in .github/workflows/webhook.yml and summary.yml)"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.repository_id}"
}

output "github_actions_workload_identity_provider" {
  description = "Set as the GCP_WORKLOAD_IDENTITY_PROVIDER GitHub Actions repo secret"
  value       = google_iam_workload_identity_pool_provider.github_actions.name
}

output "github_actions_service_account_email" {
  description = "Set as the GCP_SERVICE_ACCOUNT_EMAIL GitHub Actions repo secret"
  value       = google_service_account.github_actions_deployer.email
}
