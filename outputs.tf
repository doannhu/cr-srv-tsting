output "spanner_instance_name" {
  description = "The name of the Spanner instance"
  value       = google_spanner_instance.spanner_instance.name
}

output "spanner_instance_state" {
  description = "The state of the Spanner instance"
  value       = google_spanner_instance.spanner_instance.state
}

output "database_name" {
  description = "The name of the Spanner database"
  value       = google_spanner_database.credit_enquiry_db.name
}

output "database_ddl" {
  description = "The DDL statements used to create the database"
  value       = google_spanner_database.credit_enquiry_db.ddl
} 