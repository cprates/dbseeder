package postgres

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type Bool bool

var _ sql.Scanner = (*Bool)(nil)

// Value implements the driver.Valuer interface.
func (b *Bool) Value() (driver.Value, error) {
	return *b, nil
}

// Scan implements the sql.Scanner interface. Converts src string from Postgres into a bool to store in *b.
func (b *Bool) Scan(src any) error {
	v, ok := src.(string)
	if !ok {
		return fmt.Errorf("expect 'string', got %T", src)
	}

	fmtV := strings.ToLower(v)
	if fmtV != "yes" && fmtV != "no" {
		return fmt.Errorf("expect 'yes' or 'no', got %q", src)
	}
	*b = fmtV == "yes"

	return nil
}

type SliceInt []int

var _ sql.Scanner = (*SliceInt)(nil)

// Value implements the driver.Valuer interface.
func (s *SliceInt) Value() (driver.Value, error) {
	return *s, nil
}

// Scan implements the sql.Scanner interface. Converts src string '{1, 2, n}' from Postgres into a SliceInt to store
// in *s.
func (s *SliceInt) Scan(src any) error {
	var v string
	switch a := src.(type) {
	case string:
		v = a
	case nil:
		return nil
	default:
		return fmt.Errorf("expect 'string', got %T", src)
	}
	if v == "" {
		return nil
	}

	b := []byte(v)
	if len(b) < 2 || b[0] != '{' || b[len(b)-1] != '}' {
		return fmt.Errorf("malformed list: %s", string(b))
	}

	b[0] = '['
	b[len(b)-1] = ']'

	ints := make(SliceInt, 0)
	err := json.Unmarshal(b, &ints)
	if err != nil {
		return fmt.Errorf("marshalling %q: %w", b, err)
	}
	*s = ints

	return nil
}
