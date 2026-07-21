// Package gensql provides database-specific SQL/JS script generation for schema export.
// Supports 9 database platforms via the Driver interface. Each driver uses embedded
// text/template files for SQL generation with input validation to prevent injection.
package gensql

import (
	"fmt"
	"regexp"
	"sort"
)

// Input validation patterns for SQL injection prevention
var (
	identifierRegex  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)         // Standard SQL identifiers
	mongoDBNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_\-]*$`)       // MongoDB allows hyphens
	userRegex        = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_@.%'-]*$`)    // User names (email-like allowed)
)

// GenerateOptions configures SQL generation
type GenerateOptions struct {
	DBName     string
	SPName     string
	ExecUser   string
	Direct     bool // If true, generate direct SQL instead of stored procedure
	NoAdmin    bool // If true, skip queries requiring admin privileges
	SampleSize int  // MongoDB: documents to sample per collection (default 100)
	MaxDepth   int  // MongoDB: nested object expansion depth (default 2)
}

// Driver abstracts database-specific SQL generation. Each implementation uses
// embedded templates with variable substitution for the target database dialect.
type Driver interface {
	Name() string
	GenerateSQL(opts GenerateOptions) (string, error)
}

func ValidateUser(name string) error {
	if name == "" {
		return nil
	}
	if len(name) > 128 {
		return fmt.Errorf("user name too long (max 128 chars): %s", name)
	}
	if !userRegex.MatchString(name) {
		return fmt.Errorf("invalid user name: %s", name)
	}
	return nil
}

var drivers = map[string]Driver{
	"mysql":     &MySQLDriver{},   // MySQL 8.0+ (uses CTEs with user variables)
	"mysql57":   &MySQL57Driver{}, // MySQL 5.7 and earlier (uses GROUP_CONCAT)
	"mariadb":   &MariaDBDriver{}, // MariaDB 10.2+
	"postgres":  &PostgresDriver{},
	"sqlite":    &SQLiteDriver{},
	"mssql":     &MSSQLDriver{},     // SQL Server 2017+ (uses STRING_AGG)
	"mssql2016": &MSSQL2016Driver{}, // SQL Server 2016 and earlier (uses FOR XML PATH)
	"oracle":    &OracleDriver{},
	"mongodb":   &MongoDBDriver{}, // MongoDB 4.4+ (schema inference via sampling)
}

func ValidateIdentifier(name, label string) error {
	if name == "" {
		return fmt.Errorf("%s cannot be empty", label)
	}
	if len(name) > 128 {
		return fmt.Errorf("%s too long (max 128 chars): %s", label, name)
	}
	if !identifierRegex.MatchString(name) {
		return fmt.Errorf("invalid %s (must be alphanumeric/underscore, start with letter/underscore): %s", label, name)
	}
	return nil
}

// ValidateMongoDBName validates MongoDB database names (allows hyphens)
func ValidateMongoDBName(name string) error {
	if name == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("database name too long (max 64 chars): %s", name)
	}
	if !mongoDBNameRegex.MatchString(name) {
		return fmt.Errorf("invalid database name (must be alphanumeric/underscore/hyphen, start with letter/underscore): %s", name)
	}
	return nil
}

func GetDriver(name string) (Driver, error) {
	d, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf("unknown driver: %s (available: mysql, mysql57, mariadb, postgres, sqlite, mssql, mssql2016, oracle, mongodb)", name)
	}
	return d, nil
}

func ListDrivers() []string {
	names := make([]string, 0, len(drivers))
	for k := range drivers {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
