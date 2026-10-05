package seeder

import (
	"errors"
	"fmt"
	"strings"
)

type TableConfig struct {
	// Name is the fully qualified name of the table, e.g.: public.table0
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

type Config struct {
	// LogLevel
	// 0: disabled
	// 1: enabled
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
	// Tables is the list of all tables to be seeded
	Tables []*TableConfig
	// OutputPath is the path where to store the SQL script with the generated data. Set to 'stdout' to print to
	// stdout.
	// Defaults to seed.sql
	OutputPath string
	// DryRun generates the data without storing it in the DB when set to true
	DryRun bool
}

var (
	defaultConfig = Config{
		Schema:     "public",
		MaxRows:    100,
		MaxRetries: 3,
		OutputPath: "seed.sql",
	}
)

func (t *TableConfig) Validate(c *Config) error {
	if parts := strings.Split(t.Name, "."); len(parts) != 2 {
		return fmt.Errorf("table name %s is missing schema", t.Name)
	}

	if t.MaxRows == 0 {
		t.MaxRows = c.MaxRows
	}
	if t.MaxRows < 0 {
		return errors.New("MaxRows must be >= 0")
	}

	if t.MaxRetries == 0 {
		t.MaxRetries = c.MaxRetries
	}
	if t.MaxRetries < 0 {
		return errors.New("MaxRetries must be >= 0")
	}

	return nil
}

func (t *TableConfig) SimpleName() string {
	parts := strings.Split(t.Name, ".")
	return parts[1]
}

func (c *Config) GetTableCfg(fqName string) *TableConfig {
	for _, cfg := range c.Tables {
		if cfg.Name != fqName {
			continue
		}

		return cfg
	}

	return nil
}

// Validate validates config and sets default values.
func (c *Config) Validate() error {
	if c.Schema == "" {
		c.Schema = defaultConfig.Schema
	}

	if c.MaxRows == 0 {
		c.MaxRows = defaultConfig.MaxRows
	}
	if c.MaxRows < 0 {
		return errors.New("MaxRows must be >= 0")
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = defaultConfig.MaxRetries
	}
	if c.MaxRetries < 0 {
		return errors.New("MaxRetries must be >= 0")
	}

	if c.OutputPath == "" {
		c.OutputPath = defaultConfig.OutputPath
	}

	if len(c.Tables) == 0 {
		return errors.New("no tables configured")
	}
	for _, tableCfg := range c.Tables {
		if err := tableCfg.Validate(c); err != nil {
			return fmt.Errorf("table %s: %w", tableCfg.Name, err)
		}
	}

	return nil
}
