#!/bin/bash
#
# Useful for visually inspecting the output SQL to verify it is doing what it should
#

source ./start-fresh.sh >/dev/null

echo
echo "# Compare the columns in two schemas in the same database"
echo "# Expect SQL:"
echo "#   Add foreign key on s2.table2.table1_id"
echo "#   Drop foreign key from s2.table3.table2_id"

#
# Compare the columns in two schemas in the same database
#
./populate-db.sh db1 "
    CREATE SCHEMA s1;
    CREATE TABLE s1.table1 (
      id integer PRIMARY KEY
    );
    CREATE TABLE s1.table2 (
      id integer PRIMARY KEY,
      table1_id integer REFERENCES s1.table1(id)
    );
    CREATE TABLE s1.table3 (
      id integer, 
      table2_id integer
    );

    CREATE SCHEMA s2;
    CREATE TABLE s2.table1 (
      id integer PRIMARY KEY 
    );
    CREATE TABLE s2.table2 (
      id integer PRIMARY KEY, 
      table1_id integer 
    );
    CREATE TABLE s2.table3 (
      id integer, 
      table2_id integer REFERENCES s2.table2(id) -- This will be deleted
    );
"

echo
echo "SQL to run:"
../pgdiff -U "u1" -W "asdf" -H "localhost" -D "db1" -S "s1" -O "sslmode=disable" \
          -u "u1" -w "asdf" -h "localhost" -d "db1" -s "s2" -o "sslmode=disable" \
          FOREIGN_KEY | grep -v '^-- '

