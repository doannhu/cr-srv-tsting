-- Seed data for credit_enquiry table
-- This script is idempotent and can be run multiple times safely

MERGE INTO credit_enquiry T
USING (
  SELECT 
    'ce-1234' as credit_enquiry_id,
    'v1' as credit_enquiry_version,
    'OPEN' as enquiry_state,
    'app-9876' as application_number,
    CURRENT_TIMESTAMP() as application_start_time,
    CURRENT_TIMESTAMP() as created_time,
    CURRENT_TIMESTAMP() as updated_time
) S
ON T.credit_enquiry_id = S.credit_enquiry_id 
   AND T.credit_enquiry_version = S.credit_enquiry_version
WHEN NOT MATCHED THEN
  INSERT (
    credit_enquiry_id,
    credit_enquiry_version,
    enquiry_state,
    application_number,
    application_start_time,
    created_time,
    updated_time
  ) VALUES (
    S.credit_enquiry_id,
    S.credit_enquiry_version,
    S.enquiry_state,
    S.application_number,
    S.application_start_time,
    S.created_time,
    S.updated_time
  ); 