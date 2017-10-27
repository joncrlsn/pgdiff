#!/bin/bash
#
# Useful for visually inspecting the output SQL to verify it is doing what it should
#

source ./start-fresh.sh >/dev/null

#
# Compare the indexes in two schemas in the same database
#

./populate-db.sh db1 "
    CREATE SCHEMA s1;
    CREATE TABLE s1.table1 (
      id integer PRIMARY KEY,
      name varchar(32),
      url varchar(200)
    );
    CREATE INDEX ON s1.table1(name);

    CREATE SCHEMA s2;
    CREATE TABLE s2.table1 (
      id integer PRIMARY KEY, 
      name varchar(32),
      url varchar(200)
    );
    CREATE INDEX ON s2.table1(url);
"

echo
echo "# Compare the indexes in two schemas in the same database"
echo "# Expect SQL:"
echo "#   Add index on 2.table1.name"
echo "#   Drop index on s2.table1.url"

echo
echo "SQL to run:"
../pgdiff -U "u1" -W "asdf" -H "localhost" -D "db1" -S "s1" -O "sslmode=disable" \
          -u "u1" -w "asdf" -h "localhost" -d "db1" -s "s2" -o "sslmode=disable" \
          INDEX # | grep -v '^-- '

exit 1

#
# Compare the indexes in all schemas in two databases
#
./populate-db.sh db2 "
    CREATE SCHEMA s1;
    CREATE TABLE s1.table1 (
      id integer PRIMARY KEY
    );
    CREATE TABLE s1.table2 (
      id integer PRIMARY KEY,
      table1_id integer      -- a foreign key will be added
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
      table1_id integer REFERENCES s2.table1(id) -- This will be deleted

    );
    CREATE TABLE s2.table3 (
      id integer, 
      table2_id integer REFERENCES s2.table2(id)    
    );
"

echo
echo "# Compare the foreign keys in all schemas between two databases"
echo "# Expect SQL:"
echo "#   Add foreign key on db2.s1.table2.table1_id"
echo "#   Drop foreign key on db2.s2.table2.table1_id"

echo
echo "SQL to run:"
../pgdiff -U "u1" -W "asdf" -H "localhost" -D "db1" -S "*" -O "sslmode=disable" \
          -u "u1" -w "asdf" -h "localhost" -d "db2" -s "*" -o "sslmode=disable" \
          FOREIGN_KEY | grep -v '^-- '
