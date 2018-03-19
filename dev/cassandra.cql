CREATE KEYSPACE example WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

CREATE TABLE IF NOT EXISTS example.migration_data (
    id text,
    name text,
    PRIMARY KEY(id)
);