CREATE KEYSPACE example WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

CREATE TABLE IF NOT EXISTS example.migration_data (
    system text,
    id int,
    name text,
    description text,
    PRIMARY KEY(system, id, name)
);

CREATE TABLE IF NOT EXISTS example.migration_data_name (
    system text,
    id int,
    name text,
    description text,
    PRIMARY KEY(system, name, id)
);

CREATE TABLE IF NOT EXISTS example.migration_name (
    system text,
    name text,
    sha256sum text,
    b64enc text,
    PRIMARY KEY(system, name)
);


CREATE TABLE IF NOT EXISTS example.migration_sets (
    system text,
    name text,
    ids set<int>,
    PRIMARY KEY(system, name)
);

