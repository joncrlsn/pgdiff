#!/bin/bash
#

db=$1

PGPASSWORD=asdf psql -U u1 -h localhost -d $db >/dev/null <<EOS
    $2
EOS

