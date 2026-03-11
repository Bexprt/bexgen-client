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

-- name: CreateCategory :one
INSERT INTO categories (
    name,
    description,
    embedding
)
VALUES (
    $1,
    $2,
    $3
)
RETURNING *;

-- name: ListCategories :many
SELECT
    id,
    name,
    embedding,
    description,
    created_at,
    updated_at
FROM categories
ORDER BY name;

-- name: GetCategoryByName :one
SELECT *
FROM categories
WHERE name = $1
LIMIT 1;

-- name: CreateSubcategory :one
INSERT INTO subcategories (
    category_id,
    name,
    description,
    embedding
)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: ListSubcategories :many
SELECT
    s.id,
    s.name,
    c.name AS category,
    s.description,
    s.embedding,
    s.created_at,
    s.updated_at
FROM subcategories s
JOIN categories c
ON s.category_id = c.id
ORDER BY c.name, s.name;

-- name: ListSubcategoriesByCategory :many
SELECT
    s.id,
    s.name,
    s.description,
    s.embedding
FROM subcategories s
WHERE s.category_id = $1
ORDER BY s.name;

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = $1;

-- name: DeleteSubcategory :exec
DELETE FROM subcategories
WHERE id = $1;

-- =========================================
-- INSERT
-- =========================================

-- name: CreateSite :one
INSERT INTO sites (
    pk,
    embedding_landlord,
    embedding_site_address,
    embedding_landlord_address,
    timestamp,
    site_code,
    portfolio_type,
    channel,
    use_type,
    name,
    status,
    sprint_cascade_id,
    address,
    address2,
    city,
    state,
    zip,
    county,
    site_status,
    site_type,
    site_class,
    build_status,
    landlord,
    lease_address_2,
    lease_city,
    lease_state,
    lease_zip,
    lease_county,
    lease_vendor,
    lease_vendor_role,
    lease_vendor_address,
    lease_vendor_address2,
    lease_vendor_city,
    lease_vendor_state,
    lease_vendor_zip,
    structure_vendor,
    structure_vendor_role,
    ground_vendor,
    ground_vendor_role,
    latitude,
    longitude,
    sap,
    business_license_ids,
    landlord_reference_id
)
VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,
    $19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,
    $35,$36,$37,$38,$39,$40,$41,$42,$43,$44
)
RETURNING id;


-- =========================================
-- UPDATE ALL FIELDS
-- =========================================

-- name: UpdateSite :exec
UPDATE sites SET
    pk = $2,
    embedding_landlord = $3,
    embedding_site_address = $4,
    embedding_landlord_address = $5,
    timestamp = $6,
    site_code = $7,
    portfolio_type = $8,
    channel = $9,
    use_type = $10,
    name = $11,
    status = $12,
    sprint_cascade_id = $13,
    address = $14,
    address2 = $15,
    city = $16,
    state = $17,
    zip = $18,
    county = $19,
    site_status = $20,
    site_type = $21,
    site_class = $22,
    build_status = $23,
    landlord = $24,
    lease_address_2 = $25,
    lease_city = $26,
    lease_state = $27,
    lease_zip = $28,
    lease_county = $29,
    lease_vendor = $30,
    lease_vendor_role = $31,
    lease_vendor_address = $32,
    lease_vendor_address2 = $33,
    lease_vendor_city = $34,
    lease_vendor_state = $35,
    lease_vendor_zip = $36,
    structure_vendor = $37,
    structure_vendor_role = $38,
    ground_vendor = $39,
    ground_vendor_role = $40,
    latitude = $41,
    longitude = $42,
    sap = $43,
    business_license_ids = $44,
    landlord_reference_id = $45
WHERE id = $1;


-- =========================================
-- UPDATE EMBEDDINGS
-- =========================================

-- name: UpdateLandlordEmbedding :exec
UPDATE sites
SET embedding_landlord = $2
WHERE id = $1;


-- name: UpdateSiteAddressEmbedding :exec
UPDATE sites
SET embedding_site_address = $2
WHERE id = $1;


-- name: UpdateLandlordAddressEmbedding :exec
UPDATE sites
SET embedding_landlord_address = $2
WHERE id = $1;


-- =========================================
-- READ QUERIES (ONLY REQUESTED FIELDS)
-- =========================================

-- name: GetSiteByPK :one
SELECT
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord
FROM sites
WHERE pk = $1;


-- name: GetSitesBySiteCode :many
SELECT
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord
FROM sites
WHERE site_code = $1;


-- name: GetSitesByAddress :many
SELECT
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord
FROM sites
WHERE address = $1;


-- name: GetSitesByZip :many
SELECT
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord
FROM sites
WHERE zip = $1;


-- name: GetSitesByState :many
SELECT
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord
FROM sites
WHERE state = $1;


-- name: GetSitesByPortfolioType :many
SELECT
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord
FROM sites
WHERE portfolio_type = $1;


-- name: GetSitesBySAP :many
SELECT
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord
FROM sites
WHERE sap = $1;


-- name: GetSitesByLandlord :many
SELECT
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord
FROM sites
WHERE landlord = $1;



-- =========================================
-- VECTOR SIMILARITY SEARCH
-- =========================================

-- name: SimilarLandlord :many
SELECT
    id,
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord,
    embedding_landlord <-> $1 AS distance
FROM sites
WHERE embedding_landlord IS NOT NULL
ORDER BY embedding_landlord <-> $1
LIMIT $2;


-- name: SimilarSiteAddress :many
SELECT
    id,
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord,
    embedding_site_address <-> $1 AS distance
FROM sites
WHERE embedding_site_address IS NOT NULL
ORDER BY embedding_site_address <-> $1
LIMIT $2;


-- name: SimilarLandlordAddress :many
SELECT
    id,
    pk,
    site_code,
    address,
    zip,
    state,
    portfolio_type,
    sap,
    landlord,
    embedding_landlord_address <-> $1 AS distance
FROM sites
WHERE embedding_landlord_address IS NOT NULL
ORDER BY embedding_landlord_address <-> $1
LIMIT $2;

-- name: CreateMetadata :one
INSERT INTO metadata (
    site_id,
    document_type,
    confidence,
    document_date,
    portfolio_type,
    document_amount,
    licensed_entity,
    licensing_authority,
    document_folder,
    notes
)
VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10
)
RETURNING id;

-- name: UpdateMetadata :exec
UPDATE metadata SET
    site_id = $2,
    document_type = $3,
    confidence = $4,
    document_date = $5,
    portfolio_type = $6,
    document_amount = $7,
    licensed_entity = $8,
    licensing_authority = $9,
    document_folder = $10,
    notes = $11
WHERE id = $1;

-- name: GetMetadataByID :one
SELECT * FROM metadata
WHERE id = $1;


-- name: GetMetadataBySiteID :many
SELECT * FROM metadata
WHERE site_id = $1;


-- name: GetMetadataByDocumentType :many
SELECT * FROM metadata
WHERE document_type = $1;


-- name: GetMetadataByPortfolioType :many
SELECT * FROM metadata
WHERE portfolio_type = $1;
