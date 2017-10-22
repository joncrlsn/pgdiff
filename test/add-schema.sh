
psql -U u1 -h localhost -d db1 >/dev/null <<EOS
    $1
EOS

