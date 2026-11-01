CREATE TABLE repositories (
                              id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                              owner TEXT NOT NULL,
                              name TEXT NOT NULL,
                              full_name TEXT NOT NULL,
                              added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX repositories_full_name_lower_uidx
    ON repositories (LOWER(full_name));