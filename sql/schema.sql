-- =========================
-- UUID EXTENSION
-- =========================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- =========================
-- ENUMS
-- =========================
CREATE TYPE processing_state AS ENUM(
  'processing',
  'complete',
  'failed'
);
CREATE TYPE retry_state AS ENUM(
  'pending',
  'retried',
  'dead_letter'
);
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
  PRIMARY KEY(document_id,
  step_name)
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
ON failed_messages(retry_state,
created_at);
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
  UNIQUE(
    category_id,
    name
  )
);
CREATE INDEX idx_subcategories_category_id
ON subcategories(category_id);
-- =========================
-- SITES
-- =========================
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE sites(
  id SERIAL PRIMARY KEY,
  pk TEXT UNIQUE,
  embedding_landlord VECTOR(512),
  embedding_site_address VECTOR(512),
  embedding_landlord_address VECTOR(512),
  TIMESTAMP TIMESTAMPTZ,
  site_code TEXT,
  portfolio_type TEXT,
  channel TEXT,
  use_type TEXT,
  name TEXT,
  status TEXT,
  sprint_cascade_id TEXT,
  address TEXT,
  address2 TEXT,
  city TEXT,
  state TEXT,
  zip TEXT,
  county TEXT,
  site_status TEXT,
  site_type TEXT,
  site_class TEXT,
  build_status TEXT,
  landlord TEXT,
  lease_address_2 TEXT,
  lease_city TEXT,
  lease_state TEXT,
  lease_zip TEXT,
  lease_county TEXT,
  lease_vendor TEXT,
  lease_vendor_role TEXT,
  lease_vendor_address TEXT,
  lease_vendor_address2 TEXT,
  lease_vendor_city TEXT,
  lease_vendor_state TEXT,
  lease_vendor_zip TEXT,
  structure_vendor TEXT,
  structure_vendor_role TEXT,
  ground_vendor TEXT,
  ground_vendor_role TEXT,
  latitude DOUBLE PRECISION,
  longitude DOUBLE PRECISION,
  sap TEXT,
  business_license_ids TEXT,
  landlord_reference_id TEXT
);
CREATE INDEX idx_sites_pk
ON sites(pk);
CREATE INDEX idx_sites_site_code
ON sites(site_code);
CREATE INDEX idx_sites_address
ON sites(address);
CREATE INDEX idx_sites_zip
ON sites(zip);
CREATE INDEX idx_sites_state
ON sites(state);
CREATE INDEX idx_sites_portfolio_type
ON sites(portfolio_type);
CREATE INDEX idx_sites_sap
ON sites(sap);
CREATE INDEX idx_sites_landlord
ON sites(landlord);
CREATE INDEX idx_sites_embedding_landlord
ON sites USING ivfflat(embedding_landlord vector_cosine_ops);
CREATE INDEX idx_sites_embedding_site_address
ON sites USING ivfflat(embedding_site_address vector_cosine_ops);
CREATE INDEX idx_sites_embedding_landlord_address
ON sites USING ivfflat(embedding_landlord_address vector_cosine_ops);
-- =========================
-- METADATA
-- =========================
CREATE TABLE metadata(
  id SERIAL PRIMARY KEY,
  site_id TEXT,
  document_type TEXT,
  confidence DOUBLE PRECISION,
  document_date DATE,
  portfolio_type TEXT,
  document_amount DOUBLE PRECISION,
  licensed_entity TEXT,
  licensing_authority TEXT,
  document_folder TEXT,
  notes TEXT,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_metadata_site_id
ON metadata(site_id);
CREATE INDEX idx_metadata_document_type
ON metadata(document_type);
CREATE INDEX idx_metadata_portfolio_type
ON metadata(portfolio_type);
CREATE INDEX idx_metadata_document_date
ON metadata(document_date);
