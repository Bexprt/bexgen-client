-- =====================================
-- PROCESSING STEPS
-- =====================================

-- name: CreateProcessingStep :exec
INSERT INTO processing_steps (
    name,
    description
)
VALUES ($1, $2)
ON CONFLICT (name) DO NOTHING;


-- name: GetProcessingStep :one
SELECT *
FROM processing_steps
WHERE name = $1;


-- name: EnsureProcessingStep :exec
INSERT INTO processing_steps (name)
VALUES ($1)
ON CONFLICT (name) DO NOTHING;


-- =====================================
-- DOCUMENTS
-- =====================================

-- name: CreateDocument :one
INSERT INTO documents (
    id,
    filename,
    filepath,
    classification
)
VALUES ($1, $2, $3, $4)
RETURNING *;


-- name: GetDocumentByID :one
SELECT *
FROM documents
WHERE id = $1;


-- =====================================
-- DOCUMENT STATUS
-- =====================================

-- Upsert current step status
-- name: UpsertDocumentStatus :exec
INSERT INTO document_status (
    document_id,
    step_name,
    state,
    message
)
VALUES ($1, $2, $3, $4)
ON CONFLICT (document_id, step_name)
DO UPDATE SET
    state = EXCLUDED.state,
    message = EXCLUDED.message,
    updated_at = now();


-- name: UpdateDocumentStatus :exec
UPDATE document_status
SET
    state = $3,
    message = $4,
    updated_at = now()
WHERE document_id = $1
AND step_name = $2;


-- name: GetDocumentStatuses :many
SELECT *
FROM document_status
WHERE document_id = $1
ORDER BY updated_at DESC;


-- name: GetDocumentsByStepAndState :many
SELECT *
FROM document_status
WHERE step_name = $1
AND state = $2
ORDER BY updated_at DESC
LIMIT $3 OFFSET $4;


-- =====================================
-- FAILED MESSAGE STORAGE
-- =====================================

-- name: InsertFailedMessage :one
INSERT INTO failed_messages (
    document_id,
    topic_name,
    protobuf_payload,
    headers,
    error_message
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;


-- name: GetPendingFailedMessages :many
SELECT *
FROM failed_messages
WHERE retry_state = 'pending'
ORDER BY created_at
LIMIT $1;


-- name: MarkFailedMessageRetried :exec
UPDATE failed_messages
SET
    retry_state = 'retried',
    retry_count = retry_count + 1,
    last_retry_at = now()
WHERE id = $1;


-- name: MarkFailedMessageDeadLetter :exec
UPDATE failed_messages
SET
    retry_state = 'dead_letter',
    last_retry_at = now()
WHERE id = $1;


-- name: IncrementRetryCount :exec
UPDATE failed_messages
SET
    retry_count = retry_count + 1,
    last_retry_at = now()
WHERE id = $1;


-- =====================================
-- DASHBOARD / MONITORING
-- =====================================

-- name: CountDocumentsByStepAndState :many
SELECT
    step_name,
    state,
    COUNT(*) as total
FROM document_status
GROUP BY step_name, state;


-- name: CountFailedMessagesByState :many
SELECT
    retry_state,
    COUNT(*) as total
FROM failed_messages
GROUP BY retry_state;


-- =====================================================
-- AUDIT EVENTS QUERIES
-- =====================================================

-- name: CreateAuditEvent :one
INSERT INTO audit_events (
  resource,
  resource_id,
  action,
  actor,
  metadata
)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at;


-- name: GetAuditByResource :many
SELECT *
FROM audit_events
WHERE resource = $1
  AND resource_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;


-- name: CountAuditByResource :one
SELECT COUNT(*)
FROM audit_events
WHERE resource = $1
  AND resource_id = $2;


-- name: GetAuditTimeline :many
SELECT *
FROM audit_events
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;


-- name: GetAuditByAction :many
SELECT *
FROM audit_events
WHERE action = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;


-- name: GetAuditByActor :many
SELECT *
FROM audit_events
WHERE actor = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;


-- name: GetAuditByTimeRange :many
SELECT *
FROM audit_events
WHERE created_at BETWEEN $1 AND $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;


-- name: GetAuditByMetadata :many
SELECT *
FROM audit_events
WHERE metadata @> $1::jsonb
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;


-- name: GetAuditFiltered :many
SELECT *
FROM audit_events
WHERE ($1::text IS NULL OR resource = $1)
  AND ($2::uuid IS NULL OR resource_id = $2)
  AND ($3::text IS NULL OR action = $3)
  AND ($4::text IS NULL OR actor = $4)
  AND ($5::timestamptz IS NULL OR created_at >= $5)
  AND ($6::timestamptz IS NULL OR created_at <= $6)
ORDER BY created_at DESC
LIMIT $7 OFFSET $8;


-- name: GetAuditCursor :many
SELECT *
FROM audit_events
WHERE resource = $1
  AND resource_id = $2
  AND created_at < $3
ORDER BY created_at DESC
LIMIT $4;
