# dbseeder

`dbseeder` is a CLI tool to seed SQL databases with automatically generated data with minimal configuration by inspecting the schema to infer column types, relationships and other constraints.

Dialects supported
* PostgreSQL


## What it is not

It is not a super mega fast tool to generate gazillions of records in the blink of an eye. While it is not too slow,
speed is not the main goal.


## Install

```bash
go install github.com/cprates/dbseeder/cmd/dbseeder@latest
dbseeder seed \
  --cfg ./test_env/seed_cfg.json \
  --conn-str 'postgres://postgres:password@localhost:5432/testdb?sslmode=disable'
```


## Run from source

```bash
git clone https://github.com/cprates/dbseeder.git
cd dbseeder
go run ./cmd/dbseeder seed \
  --cfg ./test_env/seed_cfg.json \
  --conn-str 'postgres://postgres:password@localhost:5432/testdb?sslmode=disable'
```


## Try it locally

Execute the command below to start a local Postgres and seed the DB with one the commands above.
```bash
make dev_up
```

### Generated data
At the moment the generated data is relationally correct but random which may not ideal to test products.
This is my main priority at the moment but, while that is not done, one can use the `SQLFilter` described below to
work around this limitation. As an example, the configuration below limits the column `name` of `table` to any of the
names in the list, and the column `deleted_at` to a date in the given range:
```json
{
  "Name": "table0",
  "SQLFilter": "SELECT * FROM table0 WHERE name IN ('name0', 'name1', 'name2', 'name3') AND deleted_at > '2006-01-02T15:04:05' AND deleted_at <= '2026-10-02T10:00:00'"
}
```

Check the configuration below for more details and limitations.


## PostgreSQL

### Supported Constraints
* **Check Constraints**: column and table check constraints
* **Not-Null Constraints**
* **Unique Constraints**: full and partial, on single and multiple columns
* **Primary Keys**: on single and multiple columns
* **Foreign Keys**: on single and group of columns

### Supported types

| Type | What is generated |
| --- | --- |
| `boolean` | `true` or `false`. |
| `text` | A random alphanumeric string of length 0 to 20. Shorter strings are more likely. |
| `varchar(n)`, `character varying(n)` | Same alphabet, length up to `n`. |
| `char(n)`, `character(n)`, `bpchar` | Same, padded with spaces to `n`. |
| `varchar[]`, `bpchar[]` | A one-element array literal. |
| `smallint`, `integer`, `bigint` | A random value in the type's range. |
| `real`, `double precision` | A random float in the type's range. |
| `numeric` | A decimal string. Precision defaults to 10 and scale to 5 when the column does not declare them. Leading and trailing zeros are trimmed so `03.30` does not collide with `3.3` under a unique index. |
| `date` | Current time in the local location. |
| `timestamp` | Current time in UTC. |
| `timestamptz` | Current time in the local location. |
| `uuid` | A new UUID V4 string. |
| `uuid[]` | A one-element array literal holding one UUID V4. |
| `json`, `jsonb` | `{"numeric_field0": <int>}`. |
| `enum` | One of the labels defined on that enum. |

Enums are the only user-defined type that is supported. Composite types, domains, ranges, and the rest of the PostgreSQL type catalog are not registered and will fail when a column of that type is generated.

Nullable columns that are *not* part of a primary key or unique index are `NULL` about 5% of the time. Columns with a default, under the same restriction, use that default expression about 10% of the time. Primary-key and unique columns always get a generated value, never `NULL` and never the default.

Range comparisons against a constant (`>`, `>=`, `<`, `<=`, and `<>` / `!=`) narrow a value domain for `smallint`, `integer`, `bigint`, `real`, `double precision`, `numeric`, `date`, `timestamp`, and `timestamptz`. Date and timestamp domains stay inside 2000-01-01 through 2100-12-31. The same operators on `text`, `varchar`, `boolean`, `uuid`, `json`, arrays, and enums are not supported yet.


## Configuration

While the goal is to have minimal configuration, it does support a few settings

```json
{
  "LogLevel": 0,
  "Schema": "public",
  "MaxRows": 100,
  "MaxRetries": 3,
  "OutputPath": "seed.sql",
  "DryRun": false,
  "SeedAll": true,
  "ExcludeTables": [
    "table1"
  ],
  "Tables": [
    {
      "Name": "table0",
      "MaxRows": 10,
      "MaxRetries": 5,
      "SQLFilter": "SELECT * FROM table0 WHERE id > 0 AND id < 100"
    }
  ]
}
```

| Field | Default | Meaning |
| --- | --- | --- |
| `LogLevel` | `0` | `0` prints nothing. Any value above `0` prints debug information. |
| `Schema` | `public` | Schema passed to the catalog lookup. Every table in `Tables` is read from this schema. |
| `MaxRows` | `100` | Row target for every table that does not set its own. `0` or omitted means the default is used. |
| `MaxRetries` | `3` | How many times to regenerate one row after a constraint failure. `0` or omitted means the default is used". |
| `OutputPath` | `seed.sql` | File path for the seed script, or `stdout` to print to the stdout. |
| `DryRun` | `false` | `true` generates the data but does not seed the DB. |
| `SeedAll` | `true` | `true` seeds all user tables under the given schema, `false` only seeds tables in `Tables` |
| `ExcludeTables` | [] | List of table names to be excluded. Useful when SeedAll is set to true. |
| `Tables` | [] | Tables to seed if `SeedAll` is `false` or, custom configuration for specific tables if `SeedAll` is `true`. |
| `Tables[].Name` | required | Table name without the schema prefix. |
| `Tables[].MaxRows` | seeder `MaxRows` | Overrides the row target for this table only. |
| `Tables[].MaxRetries` | seeder `MaxRetries` | Overrides the retry budget for this table only. |
| `Tables[].SQLFilter` | empty | Optional SQL query that adds constraints on top of the schema to manipulate data generation. Empty means schema constraints only. |


## Manipulating data generation

At the moment the best way to manipulate data generation and express relationships between tables is by defining an
`SQLFilter` query for the target table(s).

The filter is to express constraints and relationships, not a query that is executed. The seeder does not read
existing rows to satisfy it. 

**Example 1**

Table `users` has a column `name` that has a fixed list of values it can take, and a `created_at` timestamp that must
be within an range. Define `SQLFilter` as below for `users` table.
```sql
SELECT * FROM users
WHERE
  name IN ('name0', 'name1', 'name2', 'name3')
  AND created_at > '2006-01-02T15:04:05'
  AND created_at <= '2026-10-02T10:00:00';
```

**Example 2**

Express a relationship between columns in different tables to simulate product logic. On top of `Example 1`, table
`users` column `name` must match the column `author` in table `books` and column `assigned_to_name` in table
`reviews`, and column `status` in table `reviews` must be `'IN_PROGRESS'`. Define `SQLFilter` as below for `users`
table.

```sql
SELECT * FROM users
JOIN books ON users.name = books.author
JOIN reviews ON users.name = reviews.assigned_to_name -- reviews.status = 'IN_PROGRESS' could also be added here
WHERE
  users.name IN ('name0', 'name1', 'name2', 'name3')
  AND users.created_at > '2006-01-02T15:04:05'
  AND users.created_at <= '2026-10-02T10:00:00'
  AND reviews.status = 'IN_PROGRESS';
```

## Main limitations

- Amount of records generated depend on the amount of memory of the machine running the task - all generated data is
  kept in memory during the process
- Partial Unique Indexes with conditions that use constants (e.g.: `column0 = 'val0'`) are likely to not be satisfied.
  The reason for this limitation is because values for these conditions are generated randomly unless an `SQLFilter`
  is defined to manipulate the generation.
- Self-referential foreign-keys are always set to `NULL`
- Enums are the only user-define types supported
- Range operators `>`, `>=`, `<`, `<=`, and `<>` do not support all types
  - Supported: `smallint`, `integer`, `bigint`, `real`, `double precision`, `numeric`, `date`, `timestamp`,
    `timestamptz`
  - Not supported: `text`, `varchar`, `boolean`, `uuid`, `json`
- Functions (`CONCAT`, `UPPER`, etc) are not supported at all. This includes `SQLFilter`, check constraints, partial
  unique indexes, etc
- Type casts. Note that, e.g. check constraints, that do not have a type cast originally may be converted, e.g.:
  `column_timestamptz <> NULL` => `column_timestamptz <> NULL::timestamptz`
- Catalogue names in table names are ignored - only schema is supported
- `FilterSQL` main limitations and details
  - `ANY`, CTEs (`WITH`), `HAVING`
  - Query projection, `GROUP BY`, `OFFSET`, `LIMIT`, `SORT` are ignored
  - Joins are interpreted the same way no matter their type
  - Cross joins in the form `SELECT * FROM table1, table2` are not supported as I couldn't find a use case in the
    context of this tool that couldn't be done with the other supported forms
  - `SELECT (SELECT ...);`


### SQL that filters, checks, and partial indexes can use

Check constraints and partial-index predicates essentially share the same capabilities and limitations of `SQLFilter`.


## Caveats
* If table `B` has a primary key whose check constraint allows only a few values (`id = 0 OR id = 1 OR id = 2 OR id = 3`)
  and table `A` references `B`, seeding `B` first can consume every possible key. `A` then has no parent row left to
  point at.
* Low cardinality columns, such as `varchar(1)` or a unique index over a boolean, cause many retries and slow the
  seeding process down.
* If table `B` is referenced by `A`, e.g. a foreign key, `B` is seeded once with its own constraints, and seeded
  again along with `A` constraints.


## Planned work

- Plug in generators and improved table analysis so values can follow real domains instead being completely random
- Add support for type cast operations
- Add support for serial ids
- Cut the memory consumption
- Add support for more Postgres types based on demand
- Add support for range operators on more types
- Investigate what would take to add support for basic functions like `UPPER`, `LOWER`, `CONCAT`, etc
