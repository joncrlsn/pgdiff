#!/bin/bash
#
# Drop and recreate 2 testing users (u1, u2)
# Drop and recreate 2 known databases (db1, db2) used for testing 
#

sql="
    DROP DATABASE IF EXISTS db1;
    DROP DATABASE IF EXISTS db2;

    DROP USER IF EXISTS u1;
    CREATE USER u1 WITH SUPERUSER PASSWORD 'asdf';

    CREATE DATABASE db1 WITH OWNER = u1 TEMPLATE = template0;
    CREATE DATABASE db2 WITH OWNER = u1 TEMPLATE = template0;

    DROP USER IF EXISTS u2;
    CREATE USER u2 PASSWORD 'asdf';
"

sudo su - postgres -- -c "psql <<< \"$sql\""
