DROP keyspace example;

CREATE KEYSPACE example WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

CREATE TABLE IF NOT exists example.migration_data (
  system text,
  id int,
  name text,
  description text,
  PRIMARY KEY(system, id, name)
);

CREATE TABLE IF NOT exists example.migration_data_name (
  system text,
  id int,
  name text,
  description text,
  PRIMARY KEY(system, name, id)
);

CREATE TABLE IF NOT exists example.migration_name (
  system text,
  name text,
  sha256sum text,
  b64enc text,
  PRIMARY KEY(system, name)
);

CREATE TABLE IF NOT exists example.migration_sets (
  system text,
  name text,
  ids SET<int>,
  PRIMARY KEY(system, name)
);

DROP keyspace example_scr;

CREATE KEYSPACE example_scr WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

CREATE TABLE IF NOT exists example_scr.books (
  system text,
  book_id int,
  book_name text,
  num_of_chapters int,
  book_div text,
  short_name text,
  osis_name text,
  PRIMARY KEY(system, book_id)
);

CREATE TABLE IF NOT exists example_scr.book_aliases (
  system text,
  book_id int,
  alias text,
  PRIMARY KEY(system, book_id)
);

CREATE TABLE IF NOT exists example_scr.verses (
  system text,
  book_id int,
  chapter int,
  verse_num int,
  verse_id int,
  osis_ref text,
  verse_text text,
  PRIMARY KEY(system, book_id, chapter, verse_num)
);

CREATE TABLE IF NOT exists example_scr.cross_ref (
  system text,
  verse_id int,
  verse_ref_id int,
  PRIMARY KEY(system, verse_id, verse_ref_id)
);

