# pgdiff - PostgreSQL schema diff

An important feature of this utility is that it never modifies any database directly. You are solely responsible for verifying the generated SQL *before* running it against your database.  Now that you know about that, it should give you confidence that it is safe to try out and see what SQL gets generated.

Written in GoLang, pgdiff compares the schema between two PostgreSQL databases and generates alter statements to be *manually* run against the second database.  At the moment, not everything in the schema is compared, but the things considered important are: roles, sequences, tables, columns (and their default values), primary keys, unique constraints, foreign keys, roles, ownership information, and grants. 

It is written to be easy to add and improve the accuracy of the diff.  If you find something that seems wrong and you want me to look at it, please send me two schema-only database dumps that I can test with (Use the --schema-only option with pg\_dump)

### download
[osx64](https://github.com/joncrlsn/pgrun/raw/master/bin-osx64/pgdiff "OSX 64-bit version")
[osx32](https://github.com/joncrlsn/pgrun/raw/master/bin-osx32/pgdiff "OSX version")
[linux64](https://github.com/joncrlsn/pgrun/raw/master/bin-linux64/pgdiff "Linux 64-bit version")
[linux32](https://github.com/joncrlsn/pgrun/raw/master/bin-linux32/pgdiff "Linux version")
[win64](https://github.com/joncrlsn/pgrun/raw/master/bin-win64/pgdiff.exe "Windows 64-bit version")
[win32](https://github.com/joncrlsn/pgrun/raw/master/bin-win32/pgdiff.exe "Windows version")


### usage

	pgdiff [database flags] <schemaType>


 program flags | Explanation 
-------------: | ------------------------------------
  -U1          | first db postgres user
  -pw1         | first db password
  -h1          | first db host -- default is localhost
  -p1          | first db port number.  defaults to 5432
  -d1          | first db name
  -U2          | second db postgres user
  -pw2         | second db password
  -h2          | second db host -- default is localhost
  -p2          | second db port number.  defaults to 5432
  -d2          | second db name

&lt;schemaType&gt; the type of objects in the schema to compare: ALL, ROLE, SEQUENCE, TABLE, COLUMN, INDEX, FOREIGN_KEY, OWNER, GRANT_RELATIONSHIP, GRANT_ATTRIBUTE
