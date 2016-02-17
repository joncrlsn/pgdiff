# pgdiff - PostgreSQL schema diff

pgdiff compares the schema between two PostgreSQL databases and generates alter statements to be *manually* run against the second database.  The provided pgdiff.sh script helps automate the process.  At the moment, not everything in the schema is compared, but the things considered important are: roles, sequences, tables, columns (and their default values), primary keys, unique constraints, foreign keys, roles, ownership information, and grants. 

An important feature is that pgdiff never modifies a database directly. You alone are responsible for verifying the generated SQL *before* running it against your database, so you can have confidence this is safe to try and see what SQL gets generated.

It is written to be easy to add and improve the accuracy of the diff.  If you find something that seems wrong and you want me to look at it, please send me two schema-only database dumps that I can test with (Use the --schema-only option with pg\_dump)

### download
[osx64](https://github.com/joncrlsn/pgrun/raw/master/bin-osx64/pgdiff "OSX 64-bit version")
[osx32](https://github.com/joncrlsn/pgrun/raw/master/bin-osx32/pgdiff "OSX version")
[linux64](https://github.com/joncrlsn/pgrun/raw/master/bin-linux64/pgdiff "Linux 64-bit version")
[linux32](https://github.com/joncrlsn/pgrun/raw/master/bin-linux32/pgdiff "Linux version")
[win64](https://github.com/joncrlsn/pgrun/raw/master/bin-win64/pgdiff.exe "Windows 64-bit version")
[win32](https://github.com/joncrlsn/pgrun/raw/master/bin-win32/pgdiff.exe "Windows version")


### usage

	pgdiff [options] <schemaType>


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


&lt;schemaType&gt; can be: ROLE, SEQUENCE, TABLE, COLUMN, INDEX, FOREIGN_KEY, OWNER, GRANT_RELATIONSHIP, GRANT_ATTRIBUTE
