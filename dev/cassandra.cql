CREATE KEYSPACE example WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

CREATE TABLE IF NOT EXISTS example.migration_data (
    id int,
    name text,
    PRIMARY KEY(id, name)
) WITH CLUSTERING ORDER BY (name DESC);

CREATE TABLE IF NOT EXISTS example.migration_sets (
    name text,
    ids set<int>,
    PRIMARY KEY(name)
);