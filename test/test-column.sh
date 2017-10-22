#!/bin/bash
#
# Useful for visually inspecting the output SQL to verify it is doing what it should
#

sudo su - postgres -- <<EOT
psql <<'SQL'
    DROP DATABASE IF EXISTS db1;
    DROP DATABASE IF EXISTS db2;
    DROP USER IF EXISTS u1;
    CREATE USER u1 WITH SUPERUSER PASSWORD 'asdf';
    CREATE DATABASE db1 WITH OWNER = u1 TEMPLATE = template1;
    CREATE DATABASE db2 WITH OWNER = u1 TEMPLATE = template1;
SQL
EOT
export PGPASSWORD=asdf

echo
echo "# Compare the tables in two schemas in the same database"

#
# Compare the columns in two schemas in the same database
#
psql -U u1 -h localhost -d db1 <<'EOS'
    CREATE SCHEMA s1;
    CREATE TABLE s1.table1 (
      id integer, 
      name varchar(24) 
    );

    CREATE SCHEMA s2;
    CREATE TABLE s2.table1(
                              -- id will be added to s2
      name varchar(20),       -- name will grow to 24 in s2
      description varchar(24) -- description will be dropped in s2
    );
    
EOS


echo
echo "SQL to run:"
../pgdiff -U "u1" -W "asdf" -H "localhost" -D "db1" -S "s1" -O "sslmode=disable" \
          -u "u1" -w "asdf" -h "localhost" -d "db1" -s "s2" -o "sslmode=disable" \
          COLUMN
