resource "google_billing_budget" "monthly" {
  billing_account = var.billing_account
  display_name    = "line-bot-ledger monthly budget"

  budget_filter {
    projects = ["projects/${var.project_id}"]
  }

  amount {
    specified_amount {
      currency_code = "TWD"
      units         = tostring(var.budget_amount)
    }
  }

  # threshold_percent is a fraction of var.budget_amount, so with a NT$1
  # budget these land at roughly NT$0.5 / NT$1 / NT$10 actual spend.
  threshold_rules {
    threshold_percent = 0.5
  }

  threshold_rules {
    threshold_percent = 1.0
  }

  threshold_rules {
    threshold_percent = 10.0
  }
}
