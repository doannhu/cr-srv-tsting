provider "google" {
  credentials = file(var.credentials_file)
  project     = var.project
  region      = var.region
}

resource "google_spanner_instance" "spanner_instance" {
  name         = var.instance_name
  config       = "regional-australia-southeast1"
  display_name = var.display_name
  num_nodes    = var.node_count

  labels = {
    environment = var.environment
    managed-by  = "terraform"
    service     = "credit-enquiry"
  }
}

resource "google_spanner_database" "credit_enquiry_db" {
  name     = var.database_name
  instance = google_spanner_instance.spanner_instance.name

  ddl = [
    <<EOT
    CREATE TABLE credit_enquiry (
      credit_enquiry_id STRING(37) NOT NULL,
      credit_enquiry_version STRING(50) NOT NULL,
      enquiry_state STRING(40),
      application_number STRING(37),
      application_start_time TIMESTAMP,
      created_time TIMESTAMP,
      updated_time TIMESTAMP
    ) PRIMARY KEY (credit_enquiry_id, credit_enquiry_version)
    EOT
  ]
}

# Commented out seeding-related resources
/*
resource "google_storage_bucket" "seed_scripts" {
  name     = "${var.project}-seed-scripts"
  location = "AUSTRALIA-SOUTHEAST1"

  labels = {
    environment = var.environment
    managed-by  = "terraform"
    purpose     = "seed-scripts"
  }

  uniform_bucket_level_access = true
}

resource "google_storage_bucket_iam_member" "seed_scripts_iam" {
  bucket = google_storage_bucket.seed_scripts.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${var.cloud_functions_service_account}"
}

resource "google_storage_bucket_object" "seed_script" {
  name   = "seed_credit_enquiry.sql"
  bucket = google_storage_bucket.seed_scripts.name
  source = "scripts/seed_credit_enquiry.sql"
}

resource "google_cloudfunctions_function" "seed_function" {
  name        = "seed-credit-enquiry-${var.environment}"
  description = "Function to seed credit enquiry data in ${var.environment} environment"
  runtime     = "python39"
  region      = var.region

  available_memory_mb   = 256
  source_archive_bucket = google_storage_bucket.seed_scripts.name
  source_archive_object = google_storage_bucket_object.seed_script.name
  entry_point           = "seed_data"
  timeout               = 60

  environment_variables = {
    ENVIRONMENT      = var.environment
    SPANNER_INSTANCE = google_spanner_instance.spanner_instance.name
    SPANNER_DATABASE = google_spanner_database.credit_enquiry_db.name
  }

  service_account_email = var.cloud_functions_service_account

  labels = {
    environment = var.environment
    managed-by  = "terraform"
    purpose     = "seed-data"
  }

  # Add HTTP trigger
  trigger_http = true
}

resource "google_cloudbuild_trigger" "seed_trigger" {
  name        = "seed-credit-enquiry-${var.environment}"
  description = "Trigger to seed credit enquiry data in ${var.environment} environment"
  project     = var.project_id
  location    = "global"

  github {
    owner = var.github_owner
    name  = var.github_repo
    push {
      branch = "^${var.github_branch}$"
    }
  }

  filename = "cloudbuild.yaml"
  substitutions = {
    _ENVIRONMENT = var.environment
  }

  service_account = var.cloud_build_service_account
}
*/
