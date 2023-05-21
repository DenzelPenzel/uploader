-- Create an index for fast file size calculation.
CREATE INDEX idx_records_data_length
    ON metadata(id, LENGTH(chunk));
