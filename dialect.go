package gorp

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// The Dialect interface encapsulates behaviors that differ across
// SQL databases.  At present the Dialect is only used by CreateTables()
// but this could change in the future
type Dialect interface {

	// adds a suffix to any query, usually ";"
	QuerySuffix() string

	// ToSqlType returns the SQL column type to use when creating a
	// table of the given Go Type.  maxsize can be used to switch based on
	// size.  For example, in MySQL []byte could map to BLOB, MEDIUMBLOB,
	// or LONGBLOB depending on the maxsize
	ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string

	// string to append to primary key column definitions
	AutoIncrStr() string

	// string to bind autoincrement columns to. Empty string will
	// remove reference to those columns in the INSERT statement.
	AutoIncrBindValue() string

	AutoIncrInsertSuffix(col *ColumnMap) string

	// string to append to "create table" statement for vendor specific
	// table attributes
	CreateTableSuffix() string

	// string to truncate tables
	TruncateClause() string

	// bind variable string to use when forming SQL statements
	// in many dbs it is "?", but Postgres appears to use $1
	//
	// i is a zero based index of the bind variable in this statement
	//
	BindVar(i int) string

	// Handles quoting of a field name to ensure that it doesn't raise any
	// SQL parsing exceptions by using a reserved word as a field name.
	QuoteField(field string) string

	// Handles building up of a schema.database string that is compatible with
	// the given dialect
	//
	// schema - The schema that <table> lives in
	// table - The table name
	QuotedTableForQuery(schema string, table string) string

	// Existance clause for table creation / deletion
	IfSchemaNotExists(command, schema string) string
	IfTableExists(command, schema, table string) string
	IfTableNotExists(command, schema, table string) string

	// Sql to check if an index exists
	IfIndexExists(table, index, schema string) string

	// Returns the sql to drop an index
	DropIndex(table *TableMap, index string) string

	// Handles building up of a schema.database string that is compatible with
	// the given dialect
	// table - The table that <index> is created on
	// index - The index name
	BuildIndexName(table string, index string) string
}

// IntegerAutoIncrInserter is implemented by dialects that can perform
// inserts with automatically incremented integer primary keys.  If
// the dialect can handle automatic assignment of more than just
// integers, see TargetedAutoIncrInserter.
type IntegerAutoIncrInserter interface {
	InsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error)
}

// TargetedAutoIncrInserter is implemented by dialects that can
// perform automatic assignment of any primary key type (i.e. strings
// for uuids, integers for serials, etc).
type TargetedAutoIncrInserter interface {
	// InsertAutoIncrToTarget runs an insert operation and assigns the
	// automatically generated primary key directly to the passed in
	// target.  The target should be a pointer to the primary key
	// field of the value being inserted.
	InsertAutoIncrToTarget(exec SqlExecutor, insertSql string, target interface{}, params ...interface{}) error
}

func standardInsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error) {
	res, err := exec.Exec(insertSql, params...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

///////////////////////////////////////////////////////
// sqlite3 //
/////////////

type SqliteDialect struct {
	suffix string
}

func (d SqliteDialect) QuerySuffix() string { return ";" }

func (d SqliteDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Ptr:
		return d.ToSqlType(val.Elem(), maxsize, isAutoIncr)
	case reflect.Bool:
		return "integer"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float64, reflect.Float32:
		return "real"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "blob"
		}
	}

	switch val.Name() {
	case "NullInt64":
		return "integer"
	case "NullFloat64":
		return "real"
	case "NullBool":
		return "integer"
	case "Time":
		return "datetime"
	}

	if maxsize < 1 {
		maxsize = 255
	}
	return fmt.Sprintf("varchar(%d)", maxsize)
}

// Returns autoincrement
func (d SqliteDialect) AutoIncrStr() string {
	return "autoincrement"
}

func (d SqliteDialect) AutoIncrBindValue() string {
	return "null"
}

func (d SqliteDialect) AutoIncrInsertSuffix(col *ColumnMap) string {
	return ""
}

// Returns suffix
func (d SqliteDialect) CreateTableSuffix() string {
	return d.suffix
}

// With sqlite, there technically isn't a TRUNCATE statement,
// but a DELETE FROM uses a truncate optimization:
// http://www.sqlite.org/lang_delete.html
func (d SqliteDialect) TruncateClause() string {
	return "delete from"
}

// Returns "?"
func (d SqliteDialect) BindVar(i int) string {
	return "?"
}

func (d SqliteDialect) InsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error) {
	return standardInsertAutoIncr(exec, insertSql, params...)
}

func (d SqliteDialect) QuoteField(f string) string {
	return `"` + f + `"`
}

// sqlite does not have schemas like PostgreSQL does, so just escape it like normal
func (d SqliteDialect) QuotedTableForQuery(schema string, table string) string {
	return d.QuoteField(table)
}

func (d SqliteDialect) QuotedIndex(table string, index string) string {
	return d.QuoteField(index)
}

func (d SqliteDialect) IfSchemaNotExists(command, schema string) string {
	return fmt.Sprintf("%s if not exists", command)
}

func (d SqliteDialect) IfTableExists(command, schema, table string) string {
	return fmt.Sprintf("%s if exists", command)
}

func (d SqliteDialect) IfTableNotExists(command, schema, table string) string {
	return fmt.Sprintf("%s if not exists", command)
}

func (d SqliteDialect) IfIndexExists(table, index, schema string) string {
	panic("IfIndexExists not implemented for SqliteDialect")
	return "Not Implemented"
}

// Handles building up of a schema.database string that is compatible with
// the given dialect
// table - The table that <index> is created on
// index - The index name
func (d SqliteDialect) BuildIndexName(table string, index string) string {
	panic("BuildIndexName not implemented for SqliteDialect")
	return "Not Implemented"
}

func (d SqliteDialect) DropIndex(table *TableMap, index string) string {
	sql := "drop index " + index + " " + d.QuotedIndex(table.SchemaName, index)
	return sql
}

///////////////////////////////////////////////////////
// PostgreSQL //
////////////////

type PostgresDialect struct {
	suffix string
}

func (d PostgresDialect) QuerySuffix() string { return ";" }

func (d PostgresDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Ptr:
		return d.ToSqlType(val.Elem(), maxsize, isAutoIncr)
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		if isAutoIncr {
			return "serial"
		}
		return "integer"
	case reflect.Int64, reflect.Uint64:
		if isAutoIncr {
			return "bigserial"
		}
		return "bigint"
	case reflect.Float64:
		return "double precision"
	case reflect.Float32:
		return "real"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "bytea"
		}
	}

	switch val.Name() {
	case "NullInt64":
		return "bigint"
	case "NullFloat64":
		return "double precision"
	case "NullBool":
		return "boolean"
	case "Time", "NullTime":
		return "timestamp with time zone"
	}

	if maxsize < 1 {
		maxsize = 255
	}
	return fmt.Sprintf("varchar(%d)", maxsize)
}

// Returns empty string
func (d PostgresDialect) AutoIncrStr() string {
	return ""
}

func (d PostgresDialect) AutoIncrBindValue() string {
	return "default"
}

func (d PostgresDialect) AutoIncrInsertSuffix(col *ColumnMap) string {
	return " returning " + col.ColumnName
}

// Returns suffix
func (d PostgresDialect) CreateTableSuffix() string {
	return d.suffix
}

func (d PostgresDialect) TruncateClause() string {
	return "truncate"
}

// Returns "$(i+1)"
func (d PostgresDialect) BindVar(i int) string {
	return fmt.Sprintf("$%d", i+1)
}

func (d PostgresDialect) InsertAutoIncrToTarget(exec SqlExecutor, insertSql string, target interface{}, params ...interface{}) error {
	rows, err := exec.query(insertSql, params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("No serial value returned for insert: %s Encountered error: %s", insertSql, rows.Err())
	}
	if err := rows.Scan(target); err != nil {
		return err
	}
	if rows.Next() {
		return fmt.Errorf("more than two serial value returned for insert: %s", insertSql)
	}
	return rows.Err()
}

func (d PostgresDialect) QuoteField(f string) string {
	return `"` + strings.ToLower(f) + `"`
}

// gorp with indexes, added by kim: https://github.com/kimxilxyong/gorp
// QuoteString is used to quote strings used in a WHERE clause
func (d PostgresDialect) QuoteString(f string) string {
	return `'` + strings.ToLower(f) + `'`
}

func (d PostgresDialect) QuotedTableForQuery(schema string, table string) string {
	if strings.TrimSpace(schema) == "" {
		return d.QuoteField(table)
	}

	return strings.ToLower(schema + "." + d.QuoteField(table))
}

func (d PostgresDialect) BuildIndexName(table string, index string) string {
	if strings.TrimSpace(table) == "" {
		return index
	}

	return "ix_" + table + "_" + index
}

func (d PostgresDialect) IfSchemaNotExists(command, schema string) string {
	return fmt.Sprintf("%s if not exists", command)
}

func (d PostgresDialect) IfTableExists(command, schema, table string) string {
	return fmt.Sprintf("%s if exists", command)
}

func (d PostgresDialect) IfTableNotExists(command, schema, table string) string {
	return fmt.Sprintf("%s if not exists", command)
}

func (d PostgresDialect) IfIndexExists(table, index, schema string) string {

	sql := `select
		    a.attname as ColumnName
		from
		    pg_class t,
		    pg_class i,
		    pg_index ix,
		    pg_attribute a,
		    pg_namespace n
		where
		    t.oid = ix.indrelid
		    and i.oid = ix.indexrelid
		    and a.attrelid = t.oid
		    and a.attnum = ANY(ix.indkey)
		    and t.relkind = 'r'
		    and t.relname = ` + d.QuoteString(table) +
		`
		    and i.relname = ` + d.QuoteString(d.BuildIndexName(table, index))

	if schema != "" {
		sql = sql + `
		    and n.nspname = ` + d.QuoteString(schema)
	}
	sql = sql +
		`
		    and n.oid = t.relnamespace
		order by
		    t.relname,
		    i.relname` + d.QuerySuffix()

	return sql
}

func (d PostgresDialect) DropIndex(table *TableMap, index string) string {
	sql := "drop index " + d.QuoteField(d.BuildIndexName(table.TableName, index))
	return sql
}

///////////////////////////////////////////////////////
// MySQL //
///////////

// Implementation of Dialect for MySQL databases.
type MySQLDialect struct {

	// Engine is the storage engine to use "InnoDB" vs "MyISAM" for example
	Engine string

	// Encoding is the character encoding to use for created tables
	Encoding string
}

func (d MySQLDialect) QuerySuffix() string { return ";" }

func (d MySQLDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Ptr:
		return d.ToSqlType(val.Elem(), maxsize, isAutoIncr)
	case reflect.Bool:
		return "boolean"
	case reflect.Int8:
		return "tinyint"
	case reflect.Uint8:
		return "tinyint unsigned"
	case reflect.Int16:
		return "smallint"
	case reflect.Uint16:
		return "smallint unsigned"
	case reflect.Int, reflect.Int32:
		return "int"
	case reflect.Uint, reflect.Uint32:
		return "int unsigned"
	case reflect.Int64:
		return "bigint"
	case reflect.Uint64:
		return "bigint unsigned"
	case reflect.Float64, reflect.Float32:
		return "double"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "mediumblob"
		}
	}

	switch val.Name() {
	case "NullInt64":
		return "bigint"
	case "NullFloat64":
		return "double"
	case "NullBool":
		return "tinyint"
	case "Time":
		return "datetime"
	}

	if maxsize < 1 {
		maxsize = 255
	}
	return fmt.Sprintf("varchar(%d)", maxsize)
}

// Returns auto_increment
func (d MySQLDialect) AutoIncrStr() string {
	return "auto_increment"
}

func (d MySQLDialect) AutoIncrBindValue() string {
	return "null"
}

func (d MySQLDialect) AutoIncrInsertSuffix(col *ColumnMap) string {
	return ""
}

// Returns engine=%s charset=%s  based on values stored on struct
func (d MySQLDialect) CreateTableSuffix() string {
	if d.Engine == "" || d.Encoding == "" {
		msg := "gorp - undefined"

		if d.Engine == "" {
			msg += " MySQLDialect.Engine"
		}
		if d.Engine == "" && d.Encoding == "" {
			msg += ","
		}
		if d.Encoding == "" {
			msg += " MySQLDialect.Encoding"
		}
		msg += ". Check that your MySQLDialect was correctly initialized when declared."
		panic(msg)
	}

	return fmt.Sprintf(" engine=%s charset=%s", d.Engine, d.Encoding)
}

func (d MySQLDialect) TruncateClause() string {
	return "truncate"
}

// Returns "?"
func (d MySQLDialect) BindVar(i int) string {
	return "?"
}

func (d MySQLDialect) InsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error) {
	return standardInsertAutoIncr(exec, insertSql, params...)
}

func (d MySQLDialect) QuoteField(f string) string {
	return "`" + f + "`"
}

func (d MySQLDialect) QuotedTableForQuery(schema string, table string) string {
	if strings.TrimSpace(schema) == "" {
		return d.QuoteField(table)
	}

	return schema + "." + d.QuoteField(table)
}

func (d MySQLDialect) QuotedIndex(table string, index string) string {
	if strings.TrimSpace(table) == "" {
		return d.QuoteField(index)
	}

	return d.QuoteField(table) + "." + d.QuoteField(index)
}

func (d MySQLDialect) IfSchemaNotExists(command, schema string) string {
	return fmt.Sprintf("%s if not exists", command)
}

func (d MySQLDialect) IfTableExists(command, schema, table string) string {
	return fmt.Sprintf("%s if exists", command)
}

func (d MySQLDialect) IfTableNotExists(command, schema, table string) string {
	return fmt.Sprintf("%s if not exists", command)
}

func (d MySQLDialect) IfIndexExists(table, index, schema string) string {
	sql := "select COLUMN_NAME as ColumnName " +
		"from INFORMATION_SCHEMA.STATISTICS " +
		"where table_name = '" + table + "' " +
		"and index_name = '" + index + "' "
	if schema != "" {
		sql = sql + "and table_schema = '" + schema + "'"
	}
	sql = sql + "order by SEQ_IN_INDEX asc"

	return sql
}

func (d MySQLDialect) DropIndex(table *TableMap, index string) string {
	sql := "drop index " + d.QuotedIndex(table.SchemaName, index)
	return sql
}

// Handles building up of a schema.database string that is compatible with
// the given dialect
// table - The table that <index> is created on
// index - The index name
func (d MySQLDialect) BuildIndexName(table string, index string) string {
	return index
}

///////////////////////////////////////////////////////
// Sql Server //
////////////////

// Implementation of Dialect for Microsoft SQL Server databases.
// Use gorp.SqlServerDialect{"2005"} for legacy datatypes.
// Tested with driver: github.com/denisenkom/go-mssqldb

type SqlServerDialect struct {

	// If set to "2005" legacy datatypes will be used
	Version string
}

func (d SqlServerDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Ptr:
		return d.ToSqlType(val.Elem(), maxsize, isAutoIncr)
	case reflect.Bool:
		return "bit"
	case reflect.Int8:
		return "tinyint"
	case reflect.Uint8:
		return "smallint"
	case reflect.Int16:
		return "smallint"
	case reflect.Uint16:
		return "int"
	case reflect.Int, reflect.Int32:
		return "int"
	case reflect.Uint, reflect.Uint32:
		return "bigint"
	case reflect.Int64:
		return "bigint"
	case reflect.Uint64:
		return "numeric(20,0)"
	case reflect.Float32:
		return "float(24)"
	case reflect.Float64:
		return "float(53)"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "varbinary"
		}
	}

	switch val.Name() {
	case "NullInt64":
		return "bigint"
	case "NullFloat64":
		return "float(53)"
	case "NullBool":
		return "bit"
	case "NullTime", "Time":
		if d.Version == "2005" {
			return "datetime"
		}
		return "datetime2"
	}

	if maxsize < 1 {
		if d.Version == "2005" {
			maxsize = 255
		} else {
			return fmt.Sprintf("nvarchar(max)")
		}
	}
	return fmt.Sprintf("nvarchar(%d)", maxsize)
}

// Returns auto_increment
func (d SqlServerDialect) AutoIncrStr() string {
	return "identity(0,1)"
}

// Empty string removes autoincrement columns from the INSERT statements.
func (d SqlServerDialect) AutoIncrBindValue() string {
	return ""
}

func (d SqlServerDialect) AutoIncrInsertSuffix(col *ColumnMap) string {
	return ""
}

func (d SqlServerDialect) CreateTableSuffix() string { return ";" }

func (d SqlServerDialect) TruncateClause() string {
	return "truncate table"
}

// Returns "?"
func (d SqlServerDialect) BindVar(i int) string {
	return "?"
}

func (d SqlServerDialect) InsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error) {
	return standardInsertAutoIncr(exec, insertSql, params...)
}

func (d SqlServerDialect) QuoteField(f string) string {
	return "[" + strings.Replace(f, "]", "]]", -1) + "]"
}

func (d SqlServerDialect) QuotedTableForQuery(schema string, table string) string {
	if strings.TrimSpace(schema) == "" {
		return d.QuoteField(table)
	}
	return d.QuoteField(schema) + "." + d.QuoteField(table)
}

func (d SqlServerDialect) QuotedIndex(table string, index string) string {
	if strings.TrimSpace(table) == "" {
		return d.QuoteField(index)
	}

	return strings.ToLower(table + "." + d.QuoteField(index))
}

func (d SqlServerDialect) QuerySuffix() string { return ";" }

func (d SqlServerDialect) IfSchemaNotExists(command, schema string) string {
	s := fmt.Sprintf("if schema_id(N'%s') is null %s", schema, command)
	return s
}

func (d SqlServerDialect) IfTableExists(command, schema, table string) string {
	var schema_clause string
	if strings.TrimSpace(schema) != "" {
		schema_clause = fmt.Sprintf("%s.", d.QuoteField(schema))
	}
	s := fmt.Sprintf("if object_id('%s%s') is not null %s", schema_clause, d.QuoteField(table), command)
	return s
}

func (d SqlServerDialect) IfTableNotExists(command, schema, table string) string {
	var schema_clause string
	if strings.TrimSpace(schema) != "" {
		schema_clause = fmt.Sprintf("%s.", schema)
	}
	s := fmt.Sprintf("if object_id('%s%s') is null %s", schema_clause, table, command)
	return s
}

func (d SqlServerDialect) IfIndexExists(table, index, schema string) string {
	panic("IfIndexExists not implemented for SqlServerDialect")
	return "Not Implemented"
}

func (d SqlServerDialect) DropIndex(table *TableMap, index string) string {
	sql := "drop index " + d.QuotedIndex(table.SchemaName, index)
	return sql
}

func (d SqlServerDialect) BuildIndexName(table string, index string) string {
	return index
}

///////////////////////////////////////////////////////
// Oracle //
///////////

// Implementation of Dialect for Oracle databases.
type OracleDialect struct{}

func (d OracleDialect) QuerySuffix() string { return "" }

func (d OracleDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Ptr:
		return d.ToSqlType(val.Elem(), maxsize, isAutoIncr)
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		if isAutoIncr {
			return "serial"
		}
		return "integer"
	case reflect.Int64, reflect.Uint64:
		if isAutoIncr {
			return "bigserial"
		}
		return "bigint"
	case reflect.Float64:
		return "double precision"
	case reflect.Float32:
		return "real"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "bytea"
		}
	}

	switch val.Name() {
	case "NullInt64":
		return "bigint"
	case "NullFloat64":
		return "double precision"
	case "NullBool":
		return "boolean"
	case "NullTime", "Time":
		return "timestamp with time zone"
	}

	if maxsize > 0 {
		return fmt.Sprintf("varchar(%d)", maxsize)
	} else {
		return "text"
	}

}

// Returns empty string
func (d OracleDialect) AutoIncrStr() string {
	return ""
}

func (d OracleDialect) AutoIncrBindValue() string {
	return "default"
}

func (d OracleDialect) AutoIncrInsertSuffix(col *ColumnMap) string {
	return " returning " + col.ColumnName
}

// Returns suffix
func (d OracleDialect) CreateTableSuffix() string {
	return ""
}

func (d OracleDialect) TruncateClause() string {
	return "truncate"
}

// Returns "$(i+1)"
func (d OracleDialect) BindVar(i int) string {
	return fmt.Sprintf(":%d", i+1)
}

func (d OracleDialect) InsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error) {
	rows, err := exec.query(insertSql, params...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		var id int64
		err := rows.Scan(&id)
		return id, err
	}

	return 0, errors.New("No serial value returned for insert: " + insertSql + " Encountered error: " + rows.Err().Error())
}

func (d OracleDialect) QuoteField(f string) string {
	return `"` + strings.ToUpper(f) + `"`
}

func (d OracleDialect) QuotedTableForQuery(schema string, table string) string {
	if strings.TrimSpace(schema) == "" {
		return d.QuoteField(table)
	}

	return schema + "." + d.QuoteField(table)
}

func (d OracleDialect) QuotedIndex(table string, index string) string {
	if strings.TrimSpace(table) == "" {
		return d.QuoteField(index)
	}

	return table + "." + d.QuoteField(index)
}

func (d OracleDialect) IfSchemaNotExists(command, schema string) string {
	return fmt.Sprintf("%s if not exists", command)
}

func (d OracleDialect) IfTableExists(command, schema, table string) string {
	return fmt.Sprintf("%s if exists", command)
}

func (d OracleDialect) IfTableNotExists(command, schema, table string) string {
	return fmt.Sprintf("%s if not exists", command)
}

func (d OracleDialect) IfIndexExists(table, index, schema string) string {
	panic("IfIndexExists not implemented for OracleDialect")
	return "Not Implemented"
}

func (d OracleDialect) DropIndex(table *TableMap, index string) string {
	sql := "drop index " + d.QuotedIndex(table.SchemaName, index)
	return sql
}

func (d OracleDialect) BuildIndexName(table string, index string) string {
	return index
}
