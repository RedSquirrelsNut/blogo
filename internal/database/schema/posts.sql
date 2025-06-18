CREATE TABLE IF NOT EXISTS posts (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  title         TEXT        NOT NULL,
  url           TEXT        NOT NULL UNIQUE,
  description   TEXT,
  published_at  DATETIME,
  feed_id       INTEGER     NOT NULL REFERENCES feeds(id) ON DELETE CASCADE
);

CREATE TRIGGER IF NOT EXISTS posts_updated_at
  AFTER UPDATE ON posts
  FOR EACH ROW
BEGIN
  UPDATE posts
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;
