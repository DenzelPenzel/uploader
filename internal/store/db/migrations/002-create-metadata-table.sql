CREATE TABLE IF NOT EXISTS metadata (
    id TEXT,
    chunk_index INTEGER,
    chunk BLOB,
    FOREIGN KEY(id) REFERENCES records(id)
);
