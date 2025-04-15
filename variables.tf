variable "project" {
  type        = string
  description = "The GCP project ID"
  validation {
    condition     = length(var.project) > 0
    error_message = "Project ID must not be empty."
  }
}

variable "region" {
  type        = string
  description = "The GCP region"
  default     = "australia-southeast1"
  validation {
    condition     = var.region == "australia-southeast1"
    error_message = "Region must be australia-southeast1."
  }
}

variable "credentials_file" {
  type        = string
  description = "Path to the Google Cloud service account key file"
  validation {
    condition     = fileexists(var.credentials_file)
    error_message = "The specified credentials file does not exist."
  }
}

variable "instance_name" {
  type        = string
  description = "Name of the Spanner instance. Must be between 4 and 30 characters and can only contain lowercase letters, numbers, and hyphens"
  validation {
    condition     = can(regex("^[a-z0-9-]{4,30}$", var.instance_name))
    error_message = "Instance name must be between 4 and 30 characters and can only contain lowercase letters, numbers, and hyphens."
  }
}

variable "display_name" {
  type        = string
  description = "Display name for the Spanner instance. Must be between 4 and 30 characters"
  validation {
    condition     = length(var.display_name) >= 4 && length(var.display_name) <= 30
    error_message = "Display name must be between 4 and 30 characters."
  }
}

variable "node_count" {
  type        = number
  description = "Number of Spanner nodes. Must be at least 1 and no more than 30"
  default     = 1
  validation {
    condition     = var.node_count >= 1 && var.node_count <= 30
    error_message = "Node count must be between 1 and 30."
  }
}

variable "database_name" {
  type        = string
  description = "Name of the Spanner database. Must be between 4 and 30 characters and can only contain lowercase letters, numbers, and hyphens"
  validation {
    condition     = can(regex("^[a-z0-9-]{4,30}$", var.database_name))
    error_message = "Database name must be between 4 and 30 characters and can only contain lowercase letters, numbers, and hyphens."
  }
}

variable "seed_data" {
  type        = bool
  description = "Whether to seed the database with initial test data"
  default     = false
}

variable "environment" {
  type        = string
  description = "Environment name (e.g., dev, staging, prod)"
  default     = "dev"
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod."
  }
}

variable "github_owner" {
  type        = string
  description = "GitHub repository owner/organization name"
  default     = "doannhu"
}

variable "github_repo" {
  type        = string
  description = "GitHub repository name"
  default     = "cr-srv-tsting"
}

variable "github_branch" {
  type        = string
  description = "GitHub branch to trigger Cloud Build on"
  default     = "grpc-server-dev-terraform-best-prac"
}

variable "spanner_instance" {
  type        = string
  description = "Name of the Spanner instance for Cloud Build"
  default     = "credit-enquiry-spanner-instance"
}

variable "spanner_database" {
  type        = string
  description = "Name of the Spanner database for Cloud Build"
  default     = "credit-enquiry-db"
}

# Service account variables
variable "cloud_functions_service_account" {
  description = "Service account email for Cloud Functions"
  type        = string
  default     = "terraform-cloudfunctions@p77133-py-bigquery-cloud-run.iam.gserviceaccount.com"
}

variable "cloud_build_service_account" {
  description = "Service account email for Cloud Build"
  type        = string
  default     = "terraform-cloudbuild@p77133-py-bigquery-cloud-run.iam.gserviceaccount.com"
}

variable "storage_service_account" {
  description = "Service account email for Cloud Storage"
  type        = string
  default     = "terraform-storage@p77133-py-bigquery-cloud-run.iam.gserviceaccount.com"
}

variable "project_id" {
  description = "The GCP project ID"
  type        = string
}
