#!/bin/bash
#

db=$1

PGPASSWORD=asdf psql -U u1 -h localhost -d $db <<EOS
    $2
EOS

