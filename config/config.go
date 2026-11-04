package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Table struct {
	// Name is the fully qualified name of the table, e.g.: table0
	Name string
	// MaxRows overrides general MaxRows
	MaxRows int
	// MaxRetries overrides general MaxRetries
	MaxRetries int
	// SQLFilter is an optional SQL query that adds extra constraints to the generated seed data on top of the
	// constraints from the schema itself, e.g.: express relationships with other tables with `JOIN`s or constrain
	// the generated values for a column with `column IN (1, 2, 3)`.
	// Check documentation for details and limitations.
	SQLFilter string
}

// Seeder is the set of configs for a DB seeder. An empty Seeder is a valid configuration with default values.
type Seeder struct {
	// LogLevel
	// 0: disabled
	// 1: enabled
	// Defaults to 0
	LogLevel int
	// Schema Tables are in. Defaults to "public" when not specified
	Schema string
	// MaxRows is the max. number of rows to seed a table with. The final result may be higher, e.g. a
	// a table referenced by other table foreign key, or lower, e.g.: tables with unique indexes and check constraints.
	// Defaults to 100
	MaxRows int
	// MaxRetries is the max. number of times a new row is regenerated after failing to satisfy a constraint. It is
	// applied per row so the maximum amount of retries seeding a table is MaxRows * MaxRetries.
	// Defaults to 3
	MaxRetries int
	// SeedAll seeds all user DBs under Schema.
	// Defaults to true
	SeedAll *bool
	// Tables is the list of all tables to be seeded. If SeedAll is true, defining a table here overwrites the default
	// configuration.
	Tables []*Table
	// ExcludeTables is a list of table names to be excluded. Useful when SeedAll is set to true.
	ExcludeTables []string
	// OutputPath is the path where to store the SQL script with the generated data. Set to 'stdout' to print to
	// stdout.
	// Defaults to seed.sql
	OutputPath string
	// DryRun generates the data without storing it in the DB when set to true.
	// Defaults to false
	DryRun bool
}

var (
	defaultConfig = Seeder{
		Schema:     "public",
		MaxRows:    100,
		MaxRetries: 3,
		OutputPath: "seed.sql",
	}
)

func Load(path string) (Seeder, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Seeder{}, fmt.Errorf("unable to read config file at %s: %w", path, err)
	}

	cfg := Seeder{}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Seeder{}, fmt.Errorf("unable to parse config: %w", err)
	}

	err = cfg.Validate()
	if err != nil {
		return Seeder{}, err
	}

	return cfg, nil
}

func (s *Seeder) SeedAllTables() bool {
	if s.SeedAll == nil {
		return true
	}

	return *s.SeedAll
}

// AddDefaultTable add a new table to Tables with name into the list of configured tables.
func (s *Seeder) AddDefaultTable(name string) {
	cfg := &Table{Name: name}
	_ = cfg.Validate(s)
	s.Tables = append(s.Tables, cfg)
}

func (t *Table) Validate(s *Seeder) error {
	if t.MaxRows == 0 {
		t.MaxRows = s.MaxRows
	}
	if t.MaxRows < 0 {
		return errors.New("MaxRows must be >= 0")
	}

	if t.MaxRetries == 0 {
		t.MaxRetries = s.MaxRetries
	}
	if t.MaxRetries < 0 {
		return errors.New("MaxRetries must be >= 0")
	}

	t.Name = s.Schema + "." + t.Name

	return nil
}

func (t *Table) SimpleName() string {
	parts := strings.Split(t.Name, ".")
	return parts[1]
}

func (c *Seeder) GetTableCfg(fqName string) *Table {
	for _, cfg := range c.Tables {
		if cfg.Name != fqName {
			continue
		}

		return cfg
	}

	return nil
}

// Validate validates config and sets default values.
func (s *Seeder) Validate() error {
	if s.Schema == "" {
		s.Schema = defaultConfig.Schema
	}

	if s.MaxRows == 0 {
		s.MaxRows = defaultConfig.MaxRows
	}
	if s.MaxRows < 0 {
		return errors.New("MaxRows must be >= 0")
	}

	if s.MaxRetries == 0 {
		s.MaxRetries = defaultConfig.MaxRetries
	}
	if s.MaxRetries < 0 {
		return errors.New("MaxRetries must be >= 0")
	}

	if s.OutputPath == "" {
		s.OutputPath = defaultConfig.OutputPath
	}

	if s.SeedAll == nil {
		b := true
		s.SeedAll = &b
	}

	for _, tableCfg := range s.Tables {
		if err := tableCfg.Validate(s); err != nil {
			return fmt.Errorf("table %s: %w", tableCfg.Name, err)
		}
	}

	for _, excludedTable := range s.ExcludeTables {
		for _, table := range s.Tables {
			if excludedTable == table.SimpleName() {
				return fmt.Errorf("table %s cannot be part of both seed and exclude lists", excludedTable)
			}
		}
	}

	return nil
}
