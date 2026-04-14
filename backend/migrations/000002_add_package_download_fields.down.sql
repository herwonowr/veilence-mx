ALTER TABLE packages DROP COLUMN IF EXISTS download_count_updated_at;
ALTER TABLE packages DROP COLUMN IF EXISTS popularity_score;
ALTER TABLE packages DROP COLUMN IF EXISTS download_count;
