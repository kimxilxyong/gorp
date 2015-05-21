@Kim 2015.05.21:
Today i added a lot of new field tags which create a new table:

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

The primary key is combined from two fields in this example (PID and Created)
It seems that a multifield primary key can only have one autoincrement (at least on MySQL, i have to test Postgres and SQLLite to be sure)

New is also the ability to create multi field indexes as on User and PostSub.

Changing the name, size of varchars and notnull is straightforward i think.

I have this tested only on MySQL for now, but my next step is to test it on Postgres.

I will would be very gratefull if anybody could test this on oracle and/or on sqlserver!

My goal is to have this additions sometime merged into the upstream github.com/go-gorp/gorp



@Kim 2015.05.16:
This is a fork of http://github.com/go-gorp/gorp
The purpose of this fork is to implement automatic index generation from tags or from go code directly
It was forked on 2015.05.16
Work for index support is not finished yet.	
Please do not use this repo, its work in progress and not even guaranteed to compile!

For original examples and demos please go to http://github.com/go-gorp/gorp