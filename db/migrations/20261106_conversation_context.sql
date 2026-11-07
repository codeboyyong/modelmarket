BEGIN;

ALTER TABLE user_messages
  ADD COLUMN IF NOT EXISTS pinned BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS user_conversation_summaries (
  id VARCHAR(64) PRIMARY KEY,
  conversation_id VARCHAR(64) NOT NULL REFERENCES user_conversations(id),
  branch_id VARCHAR(64) REFERENCES user_conversation_branches(id),
  branch_key VARCHAR(64) NOT NULL DEFAULT '',
  through_message_id VARCHAR(64) REFERENCES user_messages(id),
  summary TEXT NOT NULL,
  message_count INTEGER NOT NULL DEFAULT 0,
  token_estimate INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uq_summaries_conversation_branch UNIQUE (conversation_id, branch_key)
);

CREATE INDEX IF NOT EXISTS idx_messages_context
  ON user_messages(conversation_id, branch_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_pinned
  ON user_messages(conversation_id, branch_id, pinned)
  WHERE pinned = TRUE;

COMMIT;
