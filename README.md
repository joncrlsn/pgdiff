# pgdiff - PostgreSQL schema diff

pgdiff compares the schema between two PostgreSQL 9 databases and generates alter statements to be *manually* run against the second database to make them match.  The provided pgdiff.sh script helps automate the process.  

pgdiff is transparent in what it does, so it never modifies a database directly. You alone are responsible for verifying the generated SQL *before* running it against your database, so you can have confidence that pgdiff is safe to try and see what SQL gets generated.

pgdiff is written to be easy to improve the accuracy of the diff.  If you find something that seems wrong and you want me to look at it, please send me two schema-only database dumps that I can test with (Use the --schema-only option with pg\_dump)

### download
[osx](https://github.com/joncrlsn/pgrun/raw/master/bin-osx/pgdiff "OSX version")
[linux](https://github.com/joncrlsn/pgrun/raw/master/bin-linux/pgdiff "Linux version")
[windows](https://github.com/joncrlsn/pgrun/raw/master/bin-win/pgdiff.exe "Windows version")

### usage
	pgdiff [options] &lt;schemaType&gt;

 (where options are defined below and &lt;schemaType&gt; can be: ROLE, SEQUENCE, TABLE, COLUMN, INDEX, FUNCTION, VIEW, FOREIGN\_KEY, OWNER, GRANT\_RELATIONSHIP, GRANT\_ATTRIBUTE, TRIGGER)

I've found that there is an ideal order for running the different schema types.  For example, you'll always want to add new tables before you add new columns.  This is the order that has worked for me, however "your mileage may vary".

1. FUNCTION
1. ROLE
1. SEQUENCE
1. TABLE
1. VIEW
1. OWNER
1. COLUMN
1. INDEX
1. FOREIGN\_KEY
1. GRANT\_RELATIONSHIP
1. GRANT\_ATTRIBUTE
1. TRIGGER

### options

options           | explanation 
----------------: | ------------------------------------
  -V, --version   | prints the version of pgdiff being used
  -?, --help      | displays helpful usage information
  -U, --user1     | first postgres user
  -u, --user2     | second postgres user
  -W, --password1 | first db password
  -w, --password2 | second db password
  -H, --host1     | first db host. default is localhost
  -h, --host2     | second db host. default is localhost
  -P, --port1     | first db port number. default is 5432
  -p, --port2     | second db port number. default is 5432
  -D, --dbname1   | first db name
  -d, --dbname2   | second db name
  -O, --option1   | first db options. example: sslmode=disable
  -o, --option2   | second db options. example: sslmode=disable


### version history
1. 0.9.0 - Implemented ROLE, SEQUENCE, TABLE, COLUMN, INDEX, FOREIGN\_KEY, OWNER, GRANT\_RELATIONSHIP, GRANT\_ATTRIBUTE
1. 0.9.1 - Added VIEW, FUNCTION, and TRIGGER (Thank you, Shawn Carroll AKA SparkeyG)

### todo
1. fix SQL for adding an array column
1. allow editing of individual SQL lines after failure (this would be done in the script pgdiff.sh)
1. store failed SQL statements in an error file for later fixing and rerunning?
1. add windows script (or even better: re-write bash script in Go)
