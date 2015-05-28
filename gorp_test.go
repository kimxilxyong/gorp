package gorp

import (
	"database/sql"
	"encoding/json"

	"fmt"
	//"github.com/kimxilxyong/gorp"
	"log"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// verify interface compliance
var _ Dialect = PostgresDialect{}
var _ Dialect = MySQLDialect{}

type testable interface {
	GetId() int64
	Rand()
}

type Invoice struct {
	Id       int64
	Created  int64
	Updated  int64
	Memo     string
	PersonId int64
	IsPaid   bool
}

func (me *Invoice) GetId() int64 { return me.Id }
func (me *Invoice) Rand() {
	me.Memo = fmt.Sprintf("random %d", rand.Int63())
	me.Created = rand.Int63()
	me.Updated = rand.Int63()
}

type InvoiceTag struct {
	Id       int64 `db:"myid"`
	Created  int64 `db:"myCreated"`
	Updated  int64 `db:"date_updated"`
	Memo     string
	PersonId int64 `db:"name: person_id, index:idx_person"`
	IsPaid   bool  `db:"is_Paid"`
}

func (me *InvoiceTag) GetId() int64 { return me.Id }
func (me *InvoiceTag) Rand() {
	me.Memo = fmt.Sprintf("random %d", rand.Int63())
	me.Created = rand.Int63()
	me.Updated = rand.Int63()
}

// See: https://github.com/go-gorp/gorp/issues/175
type AliasTransientField struct {
	Id     int64  `db:"id"`
	Bar    int64  `db:"-"`
	BarStr string `db:"bar"`
}

func (me *AliasTransientField) GetId() int64 { return me.Id }
func (me *AliasTransientField) Rand() {
	me.BarStr = fmt.Sprintf("random %d", rand.Int63())
}

type OverriddenInvoice struct {
	Invoice
	Id string
}

type Person struct {
	Id      int64
	Created int64
	Updated int64
	FName   string
	LName   string
	Version int64
}

type FNameOnly struct {
	FName string
}

type InvoicePersonView struct {
	InvoiceId     int64
	PersonId      int64
	Memo          string
	FName         string
	LegacyVersion int64
}

type TableWithNull struct {
	Id      int64
	Str     sql.NullString
	Int64   sql.NullInt64
	Float64 sql.NullFloat64
	Bool    sql.NullBool
	Bytes   []byte
}

type WithIgnoredColumn struct {
	internal int64 `db:"-"`
	Id       int64
	Created  int64
}

type IdCreated struct {
	Id      int64
	Created int64
}

type IdCreatedExternal struct {
	IdCreated
	External int64
}

type WithStringPk struct {
	Id   string
	Name string
}

type CustomStringType string

type TypeConversionExample struct {
	Id         int64
	PersonJSON Person
	Name       CustomStringType
}

type PersonUInt32 struct {
	Id   uint32
	Name string
}

type PersonUInt64 struct {
	Id   uint64
	Name string
}

type PersonUInt16 struct {
	Id   uint16
	Name string
}

type WithEmbeddedStruct struct {
	Id int64
	Names
}

type WithEmbeddedStructBeforeAutoincrField struct {
	Names
	Id int64
}

type WithEmbeddedAutoincr struct {
	WithEmbeddedStruct
	MiddleName string
}

type Names struct {
	FirstName string
	LastName  string
}

type UniqueColumns struct {
	FirstName string
	LastName  string
	City      string
	ZipCode   int64
}

type SingleColumnTable struct {
	SomeId string
}

type CustomDate struct {
	time.Time
}

type WithCustomDate struct {
	Id    int64
	Added CustomDate
}

type WithNullTime struct {
	Id int64
}

type testTypeConverter struct{}

func (me testTypeConverter) ToDb(val interface{}) (interface{}, error) {

	switch t := val.(type) {
	case Person:
		b, err := json.Marshal(t)
		if err != nil {
			return "", err
		}
		return string(b), nil
	case CustomStringType:
		return string(t), nil
	case CustomDate:
		return t.Time, nil
	}

	return val, nil
}

func (me testTypeConverter) FromDb(target interface{}) (CustomScanner, bool) {

	return CustomScanner{}, false
}

func (p *Person) PreInsert(s SqlExecutor) error {
	p.Created = time.Now().UnixNano()
	p.Updated = p.Created
	if p.FName == "badname" {
		return fmt.Errorf("Invalid name: %s", p.FName)
	}
	return nil
}

func (p *Person) PostInsert(s SqlExecutor) error {
	p.LName = "postinsert"
	return nil
}

func (p *Person) PreUpdate(s SqlExecutor) error {
	p.FName = "preupdate"
	return nil
}

func (p *Person) PostUpdate(s SqlExecutor) error {
	p.LName = "postupdate"
	return nil
}

func (p *Person) PreDelete(s SqlExecutor) error {
	p.FName = "predelete"
	return nil
}

func (p *Person) PostDelete(s SqlExecutor) error {
	p.LName = "postdelete"
	return nil
}

func (p *Person) PostGet(s SqlExecutor) error {
	p.LName = "postget"
	return nil
}

type PersistentUser struct {
	Key            int32
	Id             string
	PassedTraining bool
}

func TestCrud(t *testing.T) {
	dbmap := initDbMap()
	defer dropAndClose(dbmap)

	foo := &AliasTransientField{BarStr: "some bar"}

	//os.Exit(99)
	testCrudInternal(t, dbmap, foo)
}

func testCrudInternal(t *testing.T, dbmap *DbMap, val testable) {
	table, _, err := dbmap.tableForPointer(val, false)
	if err != nil {
		t.Errorf("couldn't call TableFor: val=%v err=%v", val, err)
	}

	_, err = dbmap.Exec("delete from " + table.TableName)
	if err != nil {
		t.Errorf("couldn't delete rows from: val=%v err=%v", val, err)
	}

	// INSERT row
	_insert(dbmap, val)
	if val.GetId() == 0 {
		t.Errorf("val.GetId() was not set on INSERT")
		return
	}

	// SELECT row
	val2 := _get(dbmap, val, val.GetId())
	if !reflect.DeepEqual(val, val2) {
		t.Errorf("%v != %v", val, val2)
	}

	// UPDATE row and SELECT
	val.Rand()
	count := _update(dbmap, val)
	if count != 1 {
		t.Errorf("update 1 != %d", count)
	}
	val2 = _get(dbmap, val, val.GetId())
	if !reflect.DeepEqual(val, val2) {
		t.Errorf("%v != %v", val, val2)
	}

	// Select *
	rows, err := dbmap.Select(val, "select * from "+table.TableName)
	if err != nil {
		t.Errorf("couldn't select * from %s err=%v", table.TableName, err)
	} else if len(rows) != 1 {
		t.Errorf("unexpected row count in %s: %d", table.TableName, len(rows))
	} else if !reflect.DeepEqual(val, rows[0]) {
		t.Errorf("select * result: %v != %v", val, rows[0])
	}

	// DELETE row
	deleted := _del(dbmap, val)
	if deleted != 1 {
		t.Errorf("Did not delete row with Id: %d", val.GetId())
		return
	}

	// VERIFY deleted
	val2 = _get(dbmap, val, val.GetId())
	if val2 != nil {
		t.Errorf("Found invoice with id: %d after Delete()", val.GetId())
	}
}

type WithTime struct {
	Id   int64
	Time time.Time
}

type Times struct {
	One time.Time
	Two time.Time
}

type EmbeddedTime struct {
	Id string
	Times
}

func parseTimeOrPanic(format, date string) time.Time {
	t1, err := time.Parse(format, date)
	if err != nil {
		panic(err)
	}
	return t1
}

// TODO: re-enable next two tests when this is merged:
// https://github.com/ziutek/mymysql/pull/77
//
// This test currently fails w/MySQL b/c tz info is lost
func testWithTime(t *testing.T) {
	dbmap := initDbMap()
	defer dropAndClose(dbmap)

	t1 := parseTimeOrPanic("2006-01-02 15:04:05 -0700 MST",
		"2013-08-09 21:30:43 +0800 CST")
	w1 := WithTime{1, t1}
	_insert(dbmap, &w1)

	obj := _get(dbmap, WithTime{}, w1.Id)
	w2 := obj.(*WithTime)
	if w1.Time.UnixNano() != w2.Time.UnixNano() {
		t.Errorf("%v != %v", w1, w2)
	}
}

// See: https://github.com/go-gorp/gorp/issues/86
func testEmbeddedTime(t *testing.T) {
	dbmap := newDbMap()
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	dbmap.AddTable(EmbeddedTime{}).SetKeys(false, "Id")
	defer dropAndClose(dbmap)
	err := dbmap.CreateTables()
	if err != nil {
		t.Fatal(err)
	}

	time1 := parseTimeOrPanic("2006-01-02 15:04:05", "2013-08-09 21:30:43")

	t1 := &EmbeddedTime{Id: "abc", Times: Times{One: time1, Two: time1.Add(10 * time.Second)}}
	_insert(dbmap, t1)

	x := _get(dbmap, EmbeddedTime{}, t1.Id)
	t2, _ := x.(*EmbeddedTime)
	if t1.One.UnixNano() != t2.One.UnixNano() || t1.Two.UnixNano() != t2.Two.UnixNano() {
		t.Errorf("%v != %v", t1, t2)
	}
}

func BenchmarkNativeCrud(b *testing.B) {
	b.StopTimer()
	dbmap := initDbMapBench()
	defer dropAndClose(dbmap)
	b.StartTimer()

	insert := "insert into invoice_test (Created, Updated, Memo, PersonId) values (?, ?, ?, ?)"
	sel := "select Id, Created, Updated, Memo, PersonId from invoice_test where Id=?"
	update := "update invoice_test set Created=?, Updated=?, Memo=?, PersonId=? where Id=?"
	delete := "delete from invoice_test where Id=?"

	inv := &Invoice{0, 100, 200, "my memo", 0, false}

	for i := 0; i < b.N; i++ {
		res, err := dbmap.Db.Exec(insert, inv.Created, inv.Updated,
			inv.Memo, inv.PersonId)
		if err != nil {
			panic(err)
		}

		newid, err := res.LastInsertId()
		if err != nil {
			panic(err)
		}
		inv.Id = newid

		row := dbmap.Db.QueryRow(sel, inv.Id)
		err = row.Scan(&inv.Id, &inv.Created, &inv.Updated, &inv.Memo,
			&inv.PersonId)
		if err != nil {
			panic(err)
		}

		inv.Created = 1000
		inv.Updated = 2000
		inv.Memo = "my memo 2"
		inv.PersonId = 3000

		_, err = dbmap.Db.Exec(update, inv.Created, inv.Updated, inv.Memo,
			inv.PersonId, inv.Id)
		if err != nil {
			panic(err)
		}

		_, err = dbmap.Db.Exec(delete, inv.Id)
		if err != nil {
			panic(err)
		}
	}

}

func BenchmarkGorpCrud(b *testing.B) {
	b.StopTimer()
	dbmap := initDbMapBench()
	defer dropAndClose(dbmap)
	b.StartTimer()

	inv := &Invoice{0, 100, 200, "my memo", 0, true}
	for i := 0; i < b.N; i++ {
		err := dbmap.Insert(inv)
		if err != nil {
			panic(err)
		}

		obj, err := dbmap.Get(Invoice{}, inv.Id)
		if err != nil {
			panic(err)
		}

		inv2, ok := obj.(*Invoice)
		if !ok {
			panic(fmt.Sprintf("expected *Invoice, got: %v", obj))
		}

		inv2.Created = 1000
		inv2.Updated = 2000
		inv2.Memo = "my memo 2"
		inv2.PersonId = 3000
		_, err = dbmap.Update(inv2)
		if err != nil {
			panic(err)
		}

		_, err = dbmap.Delete(inv2)
		if err != nil {
			panic(err)
		}

	}
}

func initDbMapBench() *DbMap {
	dbmap := newDbMap()
	dbmap.Db.Exec("drop table if exists invoice_test")
	dbmap.AddTableWithName(Invoice{}, "invoice_test").SetKeys(true, "Id")
	err := dbmap.CreateTables()
	if err != nil {
		panic(err)
	}
	return dbmap
}

func initDbMap() *DbMap {
	dbmap := newDbMap()

	dbmap.AddTableWithName(AliasTransientField{}, "alias_trans_field_test").SetKeys(true, "id")

	dbmap.TypeConverter = testTypeConverter{}
	err := dbmap.DropTablesIfExists()
	if err != nil {
		panic(err)
	}
	err = dbmap.CreateTables()
	if err != nil {
		panic(err)
	}

	// See #146 and TestSelectAlias - this type is mapped to the same
	// table as IdCreated, but includes an extra field that isn't in the table
	dbmap.AddTableWithName(IdCreatedExternal{}, "id_created_test").SetKeys(true, "Id")

	return dbmap
}

func initDbMapNulls() *DbMap {
	dbmap := newDbMap()
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	dbmap.AddTable(TableWithNull{}).SetKeys(false, "Id")
	err := dbmap.CreateTables()
	if err != nil {
		panic(err)
	}
	return dbmap
}

func newDbMap() *DbMap {
	dialect, driver := dialectAndDriver()
	dbmap := &DbMap{Db: connect(driver), Dialect: dialect}
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	return dbmap
}

func dropAndClose(dbmap *DbMap) {
	dbmap.DropTablesIfExists()
	dbmap.Db.Close()
}

func connect(driver string) *sql.DB {
	dsn := os.Getenv("GORP_TEST_DSN")
	if dsn == "" {
		panic("GORP_TEST_DSN env variable is not set. Please see README.md")
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		panic("Error connecting to db: " + err.Error())
	}
	return db
}

func dialectAndDriver() (Dialect, string) {
	switch os.Getenv("GORP_TEST_DIALECT") {
	case "mysql":
		return MySQLDialect{"InnoDB", "UTF8"}, "mymysql"
	case "gomysql":
		return MySQLDialect{"InnoDB", "UTF8"}, "mysql"
	case "postgres":
		return PostgresDialect{}, "postgres"
	case "sqlite":
		return SqliteDialect{}, "sqlite3"
	}
	panic("GORP_TEST_DIALECT env variable is not set or is invalid. Please see README.md")
}

func _insert(dbmap *DbMap, list ...interface{}) {
	err := dbmap.Insert(list...)
	if err != nil {
		panic(err)
	}
}

func _update(dbmap *DbMap, list ...interface{}) int64 {
	count, err := dbmap.Update(list...)
	if err != nil {
		panic(err)
	}
	return count
}

func _del(dbmap *DbMap, list ...interface{}) int64 {
	count, err := dbmap.Delete(list...)
	if err != nil {
		panic(err)
	}

	return count
}

func _get(dbmap *DbMap, i interface{}, keys ...interface{}) interface{} {
	obj, err := dbmap.Get(i, keys...)
	if err != nil {
		panic(err)
	}

	return obj
}

func selectInt(dbmap *DbMap, query string, args ...interface{}) int64 {
	i64, err := SelectInt(dbmap, query, args...)
	if err != nil {
		panic(err)
	}

	return i64
}

func selectNullInt(dbmap *DbMap, query string, args ...interface{}) sql.NullInt64 {
	i64, err := SelectNullInt(dbmap, query, args...)
	if err != nil {
		panic(err)
	}

	return i64
}

func selectFloat(dbmap *DbMap, query string, args ...interface{}) float64 {
	f64, err := SelectFloat(dbmap, query, args...)
	if err != nil {
		panic(err)
	}

	return f64
}

func selectNullFloat(dbmap *DbMap, query string, args ...interface{}) sql.NullFloat64 {
	f64, err := SelectNullFloat(dbmap, query, args...)
	if err != nil {
		panic(err)
	}

	return f64
}

func selectStr(dbmap *DbMap, query string, args ...interface{}) string {
	s, err := SelectStr(dbmap, query, args...)
	if err != nil {
		panic(err)
	}

	return s
}

func selectNullStr(dbmap *DbMap, query string, args ...interface{}) sql.NullString {
	s, err := SelectNullStr(dbmap, query, args...)
	if err != nil {
		panic(err)
	}

	return s
}

func _rawexec(dbmap *DbMap, query string, args ...interface{}) sql.Result {
	res, err := dbmap.Exec(query, args...)
	if err != nil {
		panic(err)
	}
	return res
}

func _rawselect(dbmap *DbMap, i interface{}, query string, args ...interface{}) []interface{} {
	list, err := dbmap.Select(i, query, args...)
	if err != nil {
		panic(err)
	}
	return list
}
