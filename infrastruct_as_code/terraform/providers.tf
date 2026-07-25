terraform {
  required_version = ">= 1.5"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"
    }
  }

  backend "gcs" {
    bucket = "line-bot-503410-tfstate"
    prefix = "line-bot-ledger"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region

  # Local ADC has no inherent quota project; some APIs (billing budgets
  # among them) reject requests without one. Route quota/billing to our
  # actual project instead of the gcloud CLI's own default.
  user_project_override = true
  billing_project       = var.project_id
}
