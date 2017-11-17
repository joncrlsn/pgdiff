## OSX / Mac pgdiff instructions

These instructions will guide you through the process of generating SQL, reviewing it, and optionally running it on the target database.  It requires a familiarity with the bash shell in OSX.

1. download pgdiff-mac-\<version\>.tar.gz to your machine
1. untar it (a new directory will be created: called pgdiff)
1. cd into the new pgdiff directory
1. edit pgdiff.sh to change the db access values... or set them at runtime (i.e. USER1=joe NAME1=mydb USER2=joe NAME2=myotherdb ./pgdiff.sh)
1. run pgdiff.sh

## tar contents
* pgdiff - an OSX executable
* pgrun - an OSX executable for running SQL
* pgdiff.sh - a bash shell script to coordinate your interaction with pgdiff and pgrun

If you write a Go version of pgdiff.sh, please share it so we can include it or link to it for others to use.
