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
# Compare the tables in two schemas in the same database
#
psql -U u1 -h localhost -d db1 <<'EOS'
    CREATE SCHEMA s1;
    CREATE TABLE s1.table9 (id integer); -- to be added to s2
    CREATE TABLE s1.table10 (id integer);
    
    CREATE SCHEMA s2;
    CREATE TABLE s2.table10 (id integer);
    CREATE TABLE s2.table11 (id integer); -- will be dropped from s2
EOS


echo
echo "SQL to run:"
../pgdiff -U "u1" -W "asdf" -H "localhost" -D "db1" -S "s1" -O "sslmode=disable" \
          -u "u1" -w "asdf" -h "localhost" -d "db1" -s "s2" -o "sslmode=disable" \
          TABLE

echo
echo "# Compare the tables in all schemas in two databases"


#
# Compare the tables in all schemas in two databases
#
psql -U u1 -h localhost -d db2 <<'EOS'
    CREATE SCHEMA s1;
    CREATE TABLE s1.table9 (id integer);
    -- table10 will be added (in db1, but not db2) 

    CREATE SCHEMA s2;
    CREATE TABLE s2.table10 (id integer); 
    CREATE TABLE s2.table11 (id integer);
    CREATE TABLE s2.table12 (id integer); -- will be dropped (not in db1)
EOS

echo
echo "SQL to run:"
../pgdiff -U "u1" -W "asdf" -H "localhost" -D "db1" -S "*" -O "sslmode=disable" \
          -u "u1" -w "asdf" -h "localhost" -d "db2" -s "*" -o "sslmode=disable" \
          TABLE
