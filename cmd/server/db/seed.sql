-- Seed data for Bruno testing - mencakup semua kondisi status
-- Clean existing data
DELETE FROM email_threads;
DELETE FROM followups;
DELETE FROM tickets;
DELETE FROM users;

-- Insert user
INSERT INTO users (id, jira_account_id)
VALUES ('test-user-123', 'jira-acc-test');

-- Insert tickets
INSERT INTO tickets (jira_ticket_id, user_id)
VALUES ('10001', 'test-user-123'),
       ('10002', 'test-user-123'),
       ('10003', 'test-user-123');

-- ================================================================
-- PROJ-123 FOLLOWUPS (jira_ticket_id = 10001)
-- ================================================================

-- 1. Ongoing, not expired, with frequency & lastRunAt -> shows nextFollowUp
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-000000000001', '10001', 'test-user-123',
        'dev@example.com', NULL,
        'Follow up on PROJ-123 task',
        'Just checking in on the PROJ-123 progress.',
        NOW(), NOW() + INTERVAL '30 days',
        '0 9 * * 1', 0, FALSE, 'ongoing',
        'PROJ-123', NOW() - INTERVAL '2 days', NOW() - INTERVAL '7 days');

-- 2. Expired (past expire_date) with lastRunAt -> shows lastFollowUp
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-000000000002', '10001', 'test-user-123',
        'dev@example.com', NULL,
        'PROJ-123: Need update on API spec',
        'Please share the API spec draft.',
        NOW() - INTERVAL '14 days', NOW() - INTERVAL '2 days',
        '0 9 * * 1', 0, FALSE, 'ongoing',
        'PROJ-123', NOW() - INTERVAL '3 days', NOW() - INTERVAL '14 days');

-- 3. Has replied threads, not expired -> shows repliedAt
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-000000000003', '10001', 'test-user-123',
        'dev@example.com', NULL,
        'PROJ-123: Code review reminder',
        'Please review the PR when you get a chance.',
        NOW() - INTERVAL '5 days', NOW() + INTERVAL '25 days',
        '0 9 * * 1', 0, FALSE, 'ongoing',
        'PROJ-123', NOW() - INTERVAL '1 day', NOW() - INTERVAL '5 days');

-- 4. Replied AND expired -> repliedAt (replied takes priority over expired)
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-000000000005', '10001', 'test-user-123',
        'designer@example.com', NULL,
        'PROJ-123: Design assets needed',
        'Please upload the final design mockups.',
        NOW() - INTERVAL '20 days', NOW() - INTERVAL '5 days',
        '0 9 * * 1', 0, FALSE, 'ongoing',
        'PROJ-123', NOW() - INTERVAL '10 days', NOW() - INTERVAL '20 days');

-- 5. DB status "completed" (not expired, no replies) -> effective: ongoing
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-000000000006', '10001', 'test-user-123',
        'qa@example.com', NULL,
        'PROJ-123: QA sign-off',
        'Please confirm the QA sign-off for the release.',
        NOW() - INTERVAL '3 days', NOW() + INTERVAL '10 days',
        '', 0, FALSE, 'completed',
        'PROJ-123', NULL, NOW() - INTERVAL '3 days');

-- 6. DB status "stopped" (not expired, no replies) -> effective: ongoing
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-00000000000c', '10001', 'test-user-123',
        'pm@example.com', NULL,
        'PROJ-123: Old followup (stopped)',
        'This followup was stopped.',
        NOW() - INTERVAL '10 days', NOW() + INTERVAL '5 days',
        '0 9 * * 1', 0, FALSE, 'stopped',
        'PROJ-123', NULL, NOW() - INTERVAL '10 days');

-- 7. Ongoing, no frequency, no lastRunAt -> no nextFollowUp, no lastFollowUp
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-00000000000d', '10001', 'test-user-123',
        'stakeholder@example.com', NULL,
        'PROJ-123: One-time reminder',
        'This is a one-time reminder with no repeat frequency.',
        NOW(), NOW() + INTERVAL '7 days',
        '', 1, FALSE, 'ongoing',
        'PROJ-123', NULL, NOW());

-- 8. Expired without lastRunAt -> no lastFollowUp
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-00000000000e', '10001', 'test-user-123',
        'old@example.com', NULL,
        'PROJ-123: Old expired',
        'This followup expired without ever running.',
        NOW() - INTERVAL '30 days', NOW() - INTERVAL '10 days',
        '0 9 * * 1', 0, FALSE, 'ongoing',
        'PROJ-123', NULL, NOW() - INTERVAL '30 days');

-- ================================================================
-- PROJ-456 FOLLOWUPS (jira_ticket_id = 10002)
-- ================================================================

-- 9. Ongoing with frequency & lastRunAt -> shows nextFollowUp
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-000000000004', '10002', 'test-user-123',
        'pm@example.com', NULL,
        'PROJ-456: Status update',
        'What is the status of PROJ-456?',
        NOW(), NOW() + INTERVAL '30 days',
        '0 9 * * 1', 0, FALSE, 'ongoing',
        'PROJ-456', NOW() - INTERVAL '1 day', NOW() - INTERVAL '3 days');

-- 10. Ongoing no frequency -> no nextFollowUp
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-000000000007', '10002', 'test-user-123',
        'pm@example.com', NULL,
        'PROJ-456: No frequency',
        'This followup has no repeat frequency.',
        NOW(), NOW() + INTERVAL '14 days',
        '', 0, FALSE, 'ongoing',
        'PROJ-456', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day');

-- 11. Expired without lastRunAt -> no lastFollowUp
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-000000000008', '10002', 'test-user-123',
        'pm@example.com', NULL,
        'PROJ-456: Expired no run',
        'This followup expired without ever running.',
        NOW() - INTERVAL '30 days', NOW() - INTERVAL '5 days',
        '0 9 * * 1', 0, FALSE, 'ongoing',
        'PROJ-456', NULL, NOW() - INTERVAL '30 days');

-- ================================================================
-- PROJ-789 FOLLOWUPS (jira_ticket_id = 10003)
-- ================================================================

-- 12. Ongoing with frequency, no threads at all -> shows nextFollowUp
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-000000000009', '10003', 'test-user-123',
        'new@example.com', NULL,
        'PROJ-789: Welcome',
        'Welcome to the PROJ-789 team!',
        NOW(), NOW() + INTERVAL '60 days',
        '0 9 * * 1', 0, FALSE, 'ongoing',
        'PROJ-789', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day');

-- 13. Replied with multiple threads for PROJ-789 -> shows latest repliedAt
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-00000000000a', '10003', 'test-user-123',
        'team@example.com', NULL,
        'PROJ-789: Sprint review',
        'Please prepare the sprint review demo.',
        NOW() - INTERVAL '7 days', NOW() + INTERVAL '20 days',
        '0 9 * * 1', 0, FALSE, 'ongoing',
        'PROJ-789', NOW() - INTERVAL '2 days', NOW() - INTERVAL '7 days');

-- 14. Expired with lastRunAt for PROJ-789 -> shows lastFollowUp
INSERT INTO followups (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
                       start_date_time, expire_date_time, frequency, repeat,
                       followup_confirmation, status, jira_ticket_key, last_run_at, created_at)
VALUES ('a0000000-0000-0000-0000-00000000000b', '10003', 'test-user-123',
        'archived@example.com', NULL,
        'PROJ-789: Old task followup',
        'This is from an old task in PROJ-789.',
        NOW() - INTERVAL '20 days', NOW() - INTERVAL '3 days',
        '0 9 * * 1', 0, FALSE, 'ongoing',
        'PROJ-789', NOW() - INTERVAL '5 days', NOW() - INTERVAL '20 days');

-- ================================================================
-- EMAIL THREADS
-- ================================================================

-- Threads for followup 3 (PROJ-123, replied, not expired)
INSERT INTO email_threads (id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at)
VALUES ('b0000000-0000-0000-0000-000000000001', 'test-user-123',
        'a0000000-0000-0000-0000-000000000003', 'gmail-thread-1', '10001', 'replied', NOW() - INTERVAL '1 hour');

INSERT INTO email_threads (id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at)
VALUES ('b0000000-0000-0000-0000-000000000002', 'test-user-123',
        'a0000000-0000-0000-0000-000000000003', 'gmail-thread-2', '10001', 'replied', NOW() - INTERVAL '2 hours');

-- Thread for followup 1 (PROJ-123, ongoing, not replied)
INSERT INTO email_threads (id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at)
VALUES ('b0000000-0000-0000-0000-000000000003', 'test-user-123',
        'a0000000-0000-0000-0000-000000000001', 'gmail-thread-3', '10001', 'open', NOW() - INTERVAL '1 day');

-- Thread for followup 5 (PROJ-123, replied AND expired)
INSERT INTO email_threads (id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at)
VALUES ('b0000000-0000-0000-0000-000000000004', 'test-user-123',
        'a0000000-0000-0000-0000-000000000005', 'gmail-thread-4', '10001', 'replied', NOW() - INTERVAL '3 days');

-- Threads for followup 13 (PROJ-789, replied with multiple threads)
INSERT INTO email_threads (id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at)
VALUES ('b0000000-0000-0000-0000-000000000005', 'test-user-123',
        'a0000000-0000-0000-0000-00000000000a', 'gmail-thread-5', '10003', 'replied', NOW() - INTERVAL '6 hours');

INSERT INTO email_threads (id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at)
VALUES ('b0000000-0000-0000-0000-000000000006', 'test-user-123',
        'a0000000-0000-0000-0000-00000000000a', 'gmail-thread-6', '10003', 'replied', NOW() - INTERVAL '1 day');

INSERT INTO email_threads (id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at)
VALUES ('b0000000-0000-0000-0000-000000000007', 'test-user-123',
        'a0000000-0000-0000-0000-00000000000a', 'gmail-thread-7', '10003', 'replied', NOW() - INTERVAL '3 days');
