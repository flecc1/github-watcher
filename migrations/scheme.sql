CREATE TABLE repositories (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner TEXT NOT NULL,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX repositories_full_name_lower_uidx
    ON repositories (LOWER(full_name));

CREATE TABLE repository_snapshots (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    repository_id INT NOT NULL
        REFERENCES repositories(id) ON DELETE CASCADE,

    stars INT NOT NULL,
    forks INT NOT NULL,
    has_readme BOOLEAN NOT NULL,
    open_issues INT NOT NULL,
    open_pull_requests INT NOT NULL,
    collected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    commits_last_7_days INT NOT NULL,
    last_commit_at TIMESTAMP
);

CREATE INDEX repository_snapshots_repository_collected_idx
    ON repository_snapshots (repository_id, collected_at DESC);