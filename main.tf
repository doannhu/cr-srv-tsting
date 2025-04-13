provider "google" {
  credentials = file(var.credentials_file)
  project     = var.project
  region      = var.region
}

resource "google_spanner_instance" "spanner_instance" {
  name         = var.instance_name
  config       = "regional-${var.region}"
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

resource "null_resource" "seed_credit_enquiry" {
  count = var.seed_data ? 1 : 0

  provisioner "local-exec" {
    command = <<EOT
    gcloud spanner databases execute-sql ${google_spanner_database.credit_enquiry_db.name} \
      --instance=${google_spanner_instance.spanner_instance.name} \
      --sql="INSERT INTO credit_enquiry (
        credit_enquiry_id,
        credit_enquiry_version,
        enquiry_state,
        application_number,
        application_start_time,
        created_time,
        updated_time
      ) VALUES (
        'ce-1234',
        'v1',
        'OPEN',
        'app-9876',
        CURRENT_TIMESTAMP(),
        CURRENT_TIMESTAMP(),
        CURRENT_TIMESTAMP()
      );"
    EOT
  }

  depends_on = [google_spanner_database.credit_enquiry_db]
}
