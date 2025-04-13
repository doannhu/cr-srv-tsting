variable "project" {
  type        = string
  description = "Google Cloud project ID where the Spanner instance will be created"
  validation {
    condition     = length(var.project) > 0
    error_message = "Project ID cannot be empty."
  }
}

variable "region" {
  type        = string
  description = "Google Cloud region for the Spanner instance. Must be 'regional-australia-southeast'"
  validation {
    condition     = var.region == "regional-australia-southeast"
    error_message = "Only 'regional-australia-southeast' is allowed for the region."
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
  default     = "your-org"
}

variable "github_repo" {
  type        = string
  description = "GitHub repository name"
  default     = "your-repo"
}

variable "github_branch" {
  type        = string
  description = "GitHub branch to trigger Cloud Build on"
  default     = "main"
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
