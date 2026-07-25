# infrastruct_as_code/terraform

Manages the GCP resource skeleton for `line-bot-503410`: APIs, Artifact
Registry, Firestore, Secret Manager containers, service accounts/IAM, the two
Cloud Run services, the daily Cloud Scheduler job, a budget alert, and the
Workload Identity Federation setup GitHub Actions uses to deploy
(`github_actions.tf`).

It does **not** build or push container images itself — that's
`.github/workflows/webhook.yml`/`summary.yml`, triggered by a push to
`main`. Those workflows build a `linux/amd64` image (Cloud Run doesn't run
arm64) and push it straight to Artifact Registry, then deploy it — no Cloud
Build involved.

State lives in GCS (`gs://line-bot-503410-tfstate`), managed by the separate
`bootstrap/` config — see `bootstrap/`. That bucket has to exist before this
config's `terraform init` can use it as a backend, so it can't be created by
this same config (chicken-and-egg); `bootstrap/` keeps its own local state
for that one resource. Run `bootstrap/` once, first, before anything here.

## First run

0. One-time, in `bootstrap/`: `terraform init && terraform apply` to create
   the GCS state bucket this config's backend points at.
1. `cp terraform.tfvars.example terraform.tfvars` and fill in `billing_account`
   (`gcloud billing accounts list`).
2. `terraform init`
3. `terraform apply`

   This first apply uses the placeholder `webhook_image`/`summary_image`
   default (`us-docker.pkg.dev/cloudrun/container/hello`), since no real image
   exists yet. That's expected — it just gets the two Cloud Run services
   running with a hello-world container so everything else (IAM, Firestore,
   Scheduler) is wired up and testable.

4. Add the real secret values (never put these in `.tf` files or tfvars):

   ```bash
   echo -n "<channel secret>" | gcloud secrets versions add line-channel-secret --data-file=-
   echo -n "<channel access token>" | gcloud secrets versions add line-channel-token --data-file=-
   ```

5. After applying `github_actions.tf`, add two GitHub repo secrets (repo
   Settings → Secrets and variables → Actions) from the Terraform outputs:

   ```bash
   terraform output -raw github_actions_workload_identity_provider  # -> GCP_WORKLOAD_IDENTITY_PROVIDER
   terraform output -raw github_actions_service_account_email       # -> GCP_SERVICE_ACCOUNT_EMAIL
   ```

6. Push to `main`. `.github/workflows/webhook.yml`/`summary.yml` build the
   real images and deploy them — no manual image build/push step needed
   after this. The `webhook_image`/`summary_image` placeholder in
   `terraform.tfvars` only matters for the very first `apply`, before any
   workflow has run.
