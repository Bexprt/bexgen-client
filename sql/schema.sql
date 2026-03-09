-- =========================
-- UUID EXTENSION
-- =========================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- =========================
-- ENUMS
-- =========================
CREATE TYPE processing_state AS ENUM('processing',
'complete',
'failed');
CREATE TYPE retry_state AS ENUM('pending',
'retried',
'dead_letter');
-- =========================
-- DOCUMENTS
-- =========================
CREATE TABLE documents(
  id UUID PRIMARY KEY,
  filename TEXT,
  filepath TEXT,
  classification TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_documents_external_id
ON documents(external_id);
-- =========================
-- PROCESSING STEPS
-- =========================
CREATE TABLE processing_steps(
  name TEXT PRIMARY KEY,
  description TEXT,
  created_at TIMESTAMPTZ DEFAULT now()
);
-- =========================
-- DOCUMENT CURRENT STATUS
-- =========================
CREATE TABLE document_status(
  document_id UUID NOT NULL REFERENCES documents(id)
ON DELETE CASCADE,
  step_name TEXT NOT NULL REFERENCES processing_steps(name)
ON DELETE CASCADE,
  state processing_state NOT NULL,
  message TEXT,
  updated_at TIMESTAMPTZ DEFAULT now(),
  PRIMARY KEY(
    document_id,
    step_name
  )
);
CREATE INDEX idx_doc_status_doc
ON document_status(document_id);
CREATE INDEX idx_doc_status_state
ON document_status(state);
-- =========================
-- FAILED MESSAGE STORAGE
-- =========================
CREATE TABLE failed_messages(
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  document_id UUID REFERENCES documents(id)
ON DELETE 
SET NULL,
topic_name TEXT NOT NULL,
protobuf_payload BYTEA NOT NULL,
headers JSONB,
error_message TEXT,
retry_count INT DEFAULT 0,
retry_state retry_state DEFAULT 'pending',
created_at TIMESTAMPTZ DEFAULT now(),
last_retry_at TIMESTAMPTZ
);
CREATE INDEX idx_failed_retry
ON failed_messages(
  retry_state,
  created_at
);
CREATE INDEX idx_failed_doc
ON failed_messages(document_id);
-- =========================
-- AUDIT EVENTS
-- =========================
CREATE TABLE audit_events(
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  resource_id UUID,
  action TEXT NOT NULL,
  resource TEXT NOT NULL,
  actor TEXT,
  -- user email or service name
metadata JSONB,
created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_audit_resource_id_created
ON audit_events(resource,
resource_id,
created_at DESC);
CREATE INDEX idx_audit_created_at
ON audit_events(created_at DESC);
CREATE INDEX idx_audit_action_created
ON audit_events(action,
created_at DESC);
CREATE INDEX idx_audit_actor_created
ON audit_events(actor,
created_at DESC);
-- JSONB GIN index for metadata search
CREATE INDEX idx_audit_metadata
ON audit_events USING GIN(metadata);
-- =========================
-- CLASSIFICATION
-- =========================
CREATE TABLE categories(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  embedding FLOAT4 []NOT NULL,
  description TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TABLE subcategories(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  category_id UUID NOT NULL REFERENCES categories(id)
ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  embedding FLOAT4 []NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE(category_id,
  name)
);
CREATE INDEX idx_subcategories_category_id
ON subcategories(category_id);
