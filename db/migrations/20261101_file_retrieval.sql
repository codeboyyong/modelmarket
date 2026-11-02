BEGIN;

CREATE TABLE IF NOT EXISTS user_file_chunks (
  id VARCHAR(64) PRIMARY KEY,
  extraction_id VARCHAR(64) NOT NULL REFERENCES user_file_extractions(id),
  asset_id VARCHAR(64) NOT NULL REFERENCES user_workbench_assets(id),
  project_id VARCHAR(64) NOT NULL REFERENCES user_projects(id),
  chunk_index INTEGER NOT NULL,
  content TEXT NOT NULL,
  char_count INTEGER NOT NULL,
  token_estimate INTEGER NOT NULL,
  embedding_model VARCHAR(255) NOT NULL,
  embedding_data TEXT NOT NULL,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uq_file_chunks_extraction_index UNIQUE (extraction_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_file_chunks_project_asset
  ON user_file_chunks(project_id, asset_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_embedding_records_source
  ON user_embedding_records(source_type, source_id);

COMMIT;
