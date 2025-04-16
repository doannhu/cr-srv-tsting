INSERT INTO credit_enquiry (
    credit_enquiry_id,
    credit_enquiry_version,
    enquiry_state,
    application_number,
    application_start_time,
    created_time,
    updated_time
) VALUES
    ('test-id-1', '1.0.0', 'NEW', 'APP-001', CURRENT_TIMESTAMP(), CURRENT_TIMESTAMP(), CURRENT_TIMESTAMP()),
    ('test-id-2', '1.0.0', 'IN_PROGRESS', 'APP-002', CURRENT_TIMESTAMP(), CURRENT_TIMESTAMP(), CURRENT_TIMESTAMP());
