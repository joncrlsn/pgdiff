#!/bin/bash
#
# Load example dump found here:
# http://postgresguide.com/setup/example.htmlhttp://postgresguide.com/setup/example.html

#curl -L -O http://cl.ly/173L141n3402/download/example.dump
sudo su - postgres -- -c "
    createdb pgguide
    pg_restore --no-owner --dbname pgguide example.dump
    psql --dbname pgguide
"
