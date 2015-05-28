gorp with indexes

This is a fork of http://github.com/go-gorp/gorp
The purpose of this fork is to implement automatic index generation from tags or from go code

How to get it:

```
go get github.com/kimxilxyong/gorp
```


Features: 
Automatic index generation from field tags
Multifield indexes
Detection of index changes
Extended tag syntax: name, index, notnull, primarykey, autoincrement, size
Optional tag anchor: "gorp" instead of "db"
Backward compatibility, does not break current code which uses standard gorp

```
Example:
// holds a single post
type Post struct {
	Id           uint64    `db:"notnull, PID, primarykey, autoincrement"`
	SecondTestID int       `db:"notnull, name: SID"`
	Created      time.Time `db:"notnull, primarykey"`
	PostDate     time.Time `db:"notnull"`
	Site         string    `db:"name: PostSite, notnull, size:50"`
	PostId       string    `db:"notnull, size:32, unique"`
	Score        int       `db:"notnull"`
	Title        string    `db:"notnull"`
	Url          string    `db:"notnull"`
	User         string    `db:"index:idx_user, size:64"`
	PostSub      string    `db:"index:idx_user, size:128"`
	UserIP       string    `db:"notnull, size:16"`
	BodyType     string    `db:"notnull, size:64"`
	Body         string    `db:"name:PostBody, size:16384"`
	Err          error     `db:"-"` // ignore this field when storing with gorp
}
```

How to use it example code for gorp with indexes:
```
github.com\kimxilxyong\intogooglego\redditFetchGorp
```

Testing has currently been done on MySQL and Postgres

The goal is to have this additions sometime merged into the upstream github.com/go-gorp/gorp
after its stable and fully tested.

For original examples and demos please go to http://github.com/go-gorp/gorp
