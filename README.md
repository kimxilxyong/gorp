# Gorp with Indexes

This is a fork of http://github.com/go-gorp/gorp
The purpose of this fork is to implement automatic index generation from tags or from go code and support for detail/child tables.

How to get it:
```
go get github.com/kimxilxyong/gorp
```

A new feature has been added as of 2015.06.17:

* Support for detail tables

```go
// holds a single post
// You can use ether db or gorp as tag
type Post struct {
	Id        uint64     `db:"notnull, PID, primarykey, autoincrement"`
	Created   time.Time  `db:"notnull"`
	Site      string     `db:"name: PostSite, notnull, size:50"`
	User      string     `db:"index:idx_user, size:64"`
...
	Err       error      `db:"ignorefield"` // ignore this field when storing with gorp
	Comments  []*Comment `db:"relation:PostId"` 
	// will create a table Comments as a detail table with foreignkey PostId
	// If you want a different name just issue a: 
	// dbmap.AddTableWithName(post.Comment{}, "comments_embedded_test")
	// after: dbmap.AddTableWithName(post.Post{}, "posts_embedded_test")
	// but before: dbmap.CreateTablesIfNotExists()
}

// holds a single comment bound to a post - this is the detail/child struct
type Comment struct {
	Id            uint64    `db:"notnull, primarykey, autoincrement"`
	PostId        uint64    `db:"notnull, index:idx_foreign_key_postid"` // points to post.id
	User          string    `db:"size:64"`
	Title         string    `db:"size:256"`
}

New functions:

	// Inserting a post also inserts all its detail records (=comments)
	p := post.NewPost()
... add some comments to the post, check out the example
	err = dbmap.InsertWithChilds(&p)
		
	rowsaffected, err = dbmap.UpdateWithChilds(&p)
		
	res, err := dbmap.GetWithChilds(post.Post{}, PrimaryKey)
	resp := res.(*post.Post)
```

How to use it, example code for gorp with indexes:
```
github.com/kimxilxyong/intogooglego/tree/master/testGorpEmbeddedStructs
```

Features: 
* Automatic index generation from field tags
* Automatic support for detail tables
* Multifield indexes
* Detection of index changes
* Extended tag syntax: name, index, notnull, primarykey, autoincrement, size
* Optional tag anchor: "gorp" instead of "db"
* Backward compatibility, does not break current code which uses standard gorp



If you want to run the unit tests:
```
github.com\kimxilxyong\gorp> test_all.bat
```

Tested with MySQL and PostgreSQL. Other databases are not supported currently.


# Go Relational Persistence
## Original Readme from http://github.com/go-gorp/gorp

The "M" is alive and well.  Given some Go structs and a database, gorp
should remove a fair amount of boilerplate busy-work from your code.

I hope that gorp saves you time, minimizes the drudgery of getting data
in and out of your database, and helps your code focus on algorithms,
not infrastructure.

* Bind struct fields to table columns via API or tag
* Support for embedded structs
* Support for transactions
* Forward engineer db schema from structs (great for unit tests)
* Pre/post insert/update/delete hooks
* Automatically generate insert/update/delete statements for a struct
* Automatic binding of auto increment PKs back to struct after insert
* Delete by primary key(s)
* Select by primary key(s)
* Optional trace sql logging
* Bind arbitrary SQL queries to a struct
* Bind slice to SELECT query results without type assertions
* Use positional or named bind parameters in custom SELECT queries
* Optional optimistic locking using a version column (for update/deletes)

## Installation

    # install the library:
    go get gopkg.in/gorp.v1
    
    // use in your .go code:
    import (
        "gopkg.in/gorp.v1"
    )

## Versioning

This project provides a stable release (v1.x tags) and a bleeding edge codebase (master).

`gopkg.in/gorp.v1` points to the latest v1.x tag. The API's for v1 are stable and shouldn't change. Development takes place at the master branch. Althought the code in master should always compile and test successfully, it might break API's. We aim to maintain backwards compatibility, but API's and behaviour might be changed to fix a bug. Also note that API's that are new in the master branch can change until released as v2.

If you want to use bleeding edge, use `github.com/go-gorp/gorp` as import path.

## API Documentation

Full godoc output from the latest v1 release is available here:

https://godoc.org/gopkg.in/gorp.v1

For the latest code in master:

https://godoc.org/github.com/go-gorp/gorp

## Supported Go versions

This package is compatible with the last 2 major versions of Go, at this time `1.3` and `1.4`.

Any earlier versions are only supported on a best effort basis and can be dropped any time.
Go has a great compatibility promise. Upgrading your program to a newer version of Go should never really be a problem.

## Quickstart

```go
package main

import (
    "database/sql"
    "gopkg.in/gorp.v1"
    _ "github.com/mattn/go-sqlite3"
    "log"
    "time"
)

func main() {
    // initialize the DbMap
    dbmap := initDb()
    defer dbmap.Db.Close()

    // delete any existing rows
    err := dbmap.TruncateTables()
    checkErr(err, "TruncateTables failed")

    // create two posts
    p1 := newPost("Go 1.1 released!", "Lorem ipsum lorem ipsum")
    p2 := newPost("Go 1.2 released!", "Lorem ipsum lorem ipsum")

    // insert rows - auto increment PKs will be set properly after the insert
    err = dbmap.Insert(&p1, &p2)
    checkErr(err, "Insert failed")

    // use convenience SelectInt
    count, err := dbmap.SelectInt("select count(*) from posts")
    checkErr(err, "select count(*) failed")
    log.Println("Rows after inserting:", count)

    // update a row
    p2.Title = "Go 1.2 is better than ever"
    count, err = dbmap.Update(&p2)
    checkErr(err, "Update failed")
    log.Println("Rows updated:", count)

    // fetch one row - note use of "post_id" instead of "Id" since column is aliased
    //
    // Postgres users should use $1 instead of ? placeholders
    // See 'Known Issues' below
    //
    err = dbmap.SelectOne(&p2, "select * from posts where post_id=?", p2.Id)
    checkErr(err, "SelectOne failed")
    log.Println("p2 row:", p2)

    // fetch all rows
    var posts []Post
    _, err = dbmap.Select(&posts, "select * from posts order by post_id")
    checkErr(err, "Select failed")
    log.Println("All rows:")
    for x, p := range posts {
        log.Printf("    %d: %v\n", x, p)
    }

    // delete row by PK
    count, err = dbmap.Delete(&p1)
    checkErr(err, "Delete failed")
    log.Println("Rows deleted:", count)

    // delete row manually via Exec
    _, err = dbmap.Exec("delete from posts where post_id=?", p2.Id)
    checkErr(err, "Exec failed")

    // confirm count is zero
    count, err = dbmap.SelectInt("select count(*) from posts")
    checkErr(err, "select count(*) failed")
    log.Println("Row count - should be zero:", count)

    log.Println("Done!")
}

type Post struct {
    // db tag lets you specify the column name if it differs from the struct field
    Id      int64  `db:"post_id"`
    Created int64
    Title   string `db:",size:50"`               // Column size set to 50
    Body    string `db:"article_body,size:1024"` // Set both column name and size
}

func newPost(title, body string) Post {
    return Post{
        Created: time.Now().UnixNano(),
        Title:   title,
        Body:    body,
    }
}

func initDb() *gorp.DbMap {
    // connect to db using standard Go database/sql API
    // use whatever database/sql driver you wish
    db, err := sql.Open("sqlite3", "/tmp/post_db.bin")
    checkErr(err, "sql.Open failed")

    // construct a gorp DbMap
    dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

    // add a table, setting the table name to 'posts' and
    // specifying that the Id property is an auto incrementing PK
    dbmap.AddTableWithName(Post{}, "posts").SetKeys(true, "Id")

    // create the table. in a production system you'd generally
    // use a migration tool, or create the tables via scripts
    err = dbmap.CreateTablesIfNotExists()
    checkErr(err, "Create tables failed")

    return dbmap
}

func checkErr(err error, msg string) {
    if err != nil {
        log.Fatalln(msg, err)
    }
}
```

## Examples

### Mapping structs to tables

First define some types:

```go
type Invoice struct {
    Id       int64
    Created  int64
    Updated  int64
    Memo     string
    PersonId int64
}

type Person struct {
    Id      int64    
    Created int64
    Updated int64
    FName   string
    LName   string
}

// Example of using tags to alias fields to column names
// The 'db' value is the column name
//
// A hyphen will cause gorp to skip this field, similar to the
// Go json package.
//
// This is equivalent to using the ColMap methods:
//
//   table := dbmap.AddTableWithName(Product{}, "product")
//   table.ColMap("Id").Rename("product_id")
//   table.ColMap("Price").Rename("unit_price")
//   table.ColMap("IgnoreMe").SetTransient(true)
//
type Product struct {
    Id         int64     `db:"product_id"`
    Price      int64     `db:"unit_price"`
    IgnoreMe   string    `db:"-"`
}
```

Then create a mapper, typically you'd do this one time at app startup:

```go
// connect to db using standard Go database/sql API
// use whatever database/sql driver you wish
db, err := sql.Open("mymysql", "tcp:localhost:3306*mydb/myuser/mypassword")

// construct a gorp DbMap
dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}

// register the structs you wish to use with gorp
// you can also use the shorter dbmap.AddTable() if you 
// don't want to override the table name
//
// SetKeys(true) means we have a auto increment primary key, which
// will get automatically bound to your struct post-insert
//
t1 := dbmap.AddTableWithName(Invoice{}, "invoice_test").SetKeys(true, "Id")
t2 := dbmap.AddTableWithName(Person{}, "person_test").SetKeys(true, "Id")
t3 := dbmap.AddTableWithName(Product{}, "product_test").SetKeys(true, "Id")


Additionally, when using Postgres as your database, you should utilize `$1` instead 
of `?` placeholders as utilizing `?` placeholders when querying Postgres will result 
in `pq: operator does not exist` errors. Alternatively, use 
`dbMap.Dialect.BindVar(varIdx)` to get the proper variable binding for your dialect.

### time.Time and time zones

gorp will pass `time.Time` fields through to the `database/sql` driver, but note that 
the behavior of this type varies across database drivers.

MySQL users should be especially cautious.  See: https://github.com/ziutek/mymysql/pull/77

To avoid any potential issues with timezone/DST, consider using an integer field for time
data and storing UNIX time.

## Running the tests

The included tests may be run against MySQL, Postgresql, or sqlite3.
You must set two environment variables so the test code knows which driver to
use, and how to connect to your database.

```sh
# MySQL example:
export GORP_TEST_DSN=gomysql_test/gomysql_test/abc123
export GORP_TEST_DIALECT=mysql

# run the tests
go test

# run the tests and benchmarks
go test -bench="Bench" -benchtime 10
```

Testing has currently been done on MySQL and Postgres

The goal is to have this additions sometime merged into the upstream github.com/go-gorp/gorp
after its stable and fully tested.

For original examples and demos please go to http://github.com/go-gorp/gorp
