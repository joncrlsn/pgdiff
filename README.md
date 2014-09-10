# pgdiff - PostgreSQL schema diff

Written in GoLang, this utility compares the schema between two PostgreSQL databases and generates alter statements to be *manually* run against the second database.  Not everything in the schema is compared, but the things considered important (at the moment) are: tables, columns (and their default values), foreign keys... and soon constraints and user roles. 

It is written to be easy to add and improve the accuracy of the diff.  Please let me know if you think this goal has not been met.  I'm very interested in suggestions and contributions to improve this program.  I'm not a GoLang expert yet, but each program I write gets me closer to that goal.

I'm a big fan of GoLang because of how easy it is to deliver a single executable on almost any platform.  But, just as important I love the design choices and the concurrency features which I've only begun to delve into.  Streaming objects back (via a channel) from a go routine is far better than returning a potentially massive list of objects.

<!-- A couple of binaries to save you the effort: [Mac](https://github.com/joncrlsn/pgdiff/raw/master/bin-osx/pgdiff "OSX version") -->

## usage

	pgdiff [database flags] 


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
