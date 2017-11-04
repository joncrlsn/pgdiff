# pgdiff - PostgreSQL schema diff

pgdiff compares the schema between two PostgreSQL 9 databases and generates alter statements to be *manually* run against the second database to make them match.  The provided pgdiff.sh script helps automate the process.  

pgdiff is transparent in what it does, so it never modifies a database directly. You alone are responsible for verifying the generated SQL before running it against your database.  Go ahead and see what SQL gets generated.

pgdiff is written to be easy to expand and improve the accuracy of the diff.


### download
[osx](https://github.com/joncrlsn/pgdiff/raw/master/bin-osx/pgdiff.tgz "OSX version") &nbsp; [linux](https://github.com/joncrlsn/pgdiff/raw/master/bin-linux/pgdiff.tgz "Linux version") &nbsp; [windows](https://github.com/joncrlsn/pgdiff/raw/master/bin-win/pgdiff.exe "Windows version")


### usage
	pgdiff [options] <schemaType>

(where options and &lt;schemaType&gt; are listed below)

I have found that there is an ideal order for running the different schema types.  This order should minimize the problems you encounter.  For example, you will always want to add new tables before you add new columns.  This is the order that has worked for me.

In addition, some types can have dependencies which are not in the right order.  A classic case is views which depend on other views.  The missing view SQL is generated in alphabetical order so if a view create fails due to a missing view, just run the views SQL file over again. The pgdiff.sh script will prompt you about running it again.
 
Schema type ordering:

1. SCHEMA
1. ROLE
1. SEQUENCE
1. TABLE
1. COLUMN
1. INDEX
1. VIEW
1. FOREIGN\_KEY
1. FUNCTION
1. TRIGGER
1. OWNER
1. GRANT\_RELATIONSHIP
1. GRANT\_ATTRIBUTE


### example
I have found it helpful to take ```--schema-only``` dumps of the databases in question, load them into a local postgres, then do my sql generation and testing there before running the SQL against a more official database. Your local postgres instance will need the correct users/roles populated because db dumps do not copy that information.

```
pgdiff -U dbuser -H localhost -D refDB  -O "sslmode=disable" \
       -u dbuser -h localhost -d compDB -o "sslmode=disable" \
       TABLE 
```


### options

options           | explanation 
----------------: | ------------------------------------
  -V, --version   | prints the version of pgdiff being used
  -?, --help      | displays helpful usage information
  -U, --user1     | first postgres user
  -u, --user2     | second postgres user
  -W, --password1 | first db password
  -w, --password2 | second db password
  -H, --host1     | first db host.  default is localhost
  -h, --host2     | second db host. default is localhost
  -P, --port1     | first db port number.  default is 5432
  -p, --port2     | second db port number. default is 5432
  -D, --dbname1   | first db name
  -d, --dbname2   | second db name
  -S, --schema1   | first schema name.  default is public
  -s, --schema2   | second schema name. default is public
  -O, --option1   | first db options. example: sslmode=disable
  -o, --option2   | second db options. example: sslmode=disable


### getting started on linux and osx

linux and osx binaries are packaged with an extra, optional bash script and pgrun program that helps speed the diffing process. 

1. download the tgz file for your OS
1. untar it:  ```tar -xzvf pgdiff.tgz```
1. cd to the new pgdiff directory
1. edit the db connection defaults in pgdiff.sh 
1. ...or manually run pgdiff for each schema type listed in the usage section above
1. review the SQL output for each schema type and, if you want to make them match, run it against db2 (Function SQL requires the use of pgrun instead of psql)


### getting started on windows

1. download pgdiff.exe from the bin-win directory on github
1. edit the db connection defaults in pgdiff.sh or...
1. manually run pgdiff for each schema type listed in the usage section above
1. review the SQL output and, if you want to make them match, run it against db2 


### version history
1. 0.9.0 - Implemented ROLE, SEQUENCE, TABLE, COLUMN, INDEX, FOREIGN\_KEY, OWNER, GRANT\_RELATIONSHIP, GRANT\_ATTRIBUTE
1. 0.9.1 - Added VIEW, FUNCTION, and TRIGGER (Thank you, Shawn Carroll AKA SparkeyG)
1. 0.9.2 - Fixed bug when using the non-default port
1. 0.9.3 - Added support for schemas other than public. Fixed VARCHAR bug when no max length specified


### todo
1. fix SQL for adding an array column
1. create windows version of pgdiff.sh (or even better: re-write it all in Go)
1. allow editing of individual SQL lines after failure (this would probably be done in the script pgdiff.sh)
1. store failed SQL statements in an error file for later fixing and rerunning?
