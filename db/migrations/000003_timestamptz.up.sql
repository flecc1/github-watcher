ALTER TABLE repositories
ALTER COLUMN added_at TYPE TIMESTAMPTZ USING added_at AT TIME ZONE 'UTC';

ALTER TABLE repository_snapshots
      ALTER COLUMN collected_at TYPE TIMESTAMPTZ USING collected_at AT TIME ZONE 'UTC',
            ALTER COLUMN last_commit_at TYPE TIMESTAMPTZ USING last_commit_at AT TIME ZONE 'UTC';
