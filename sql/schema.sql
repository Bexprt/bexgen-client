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
