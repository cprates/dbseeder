CREATE TABLE fks_table_child1 (
	id integer PRIMARY KEY,
	c1c text
);
CREATE TABLE fks_table_child2 (
	id integer PRIMARY KEY,
	c1c text
);
CREATE TABLE fks_table (
	id integer PRIMARY KEY,
	c1 integer REFERENCES fks_table_child1,
	c2 integer REFERENCES fks_table_child2
);

CREATE TABLE unique_indexes_table (
    id integer,
    c1 text,
    c2 text,
	c3 text,
	c4 text,
	c5 text PRIMARY KEY,
	c6 text UNIQUE,
	deleted_at timestamp,
	UNIQUE (c3, c4)
);
CREATE UNIQUE INDEX unique_indexes_table_idx1 ON unique_indexes_table (id);
CREATE UNIQUE INDEX unique_indexes_table_idx2 ON unique_indexes_table (id, c1);
CREATE UNIQUE INDEX unique_indexes_table_idx3 ON unique_indexes_table (deleted_at) WHERE (deleted_at != NULL AND c1 = c4);
