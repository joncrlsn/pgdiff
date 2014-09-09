package main

import "flag"
import "github.com/joncrlsn/pgutil"

func parseFlags() (pgutil.DbInfo, pgutil.DbInfo) {

	var dbUser1 = flag.String("U1", "", "db user")
	var dbPass1 = flag.String("pw1", "", "db password")
	var dbHost1 = flag.String("h1", "localhost", "db host")
	var dbPort1 = flag.Int("p1", 5432, "db port")
	var dbName1 = flag.String("d1", "", "db name")
	var dbOptions1 = flag.String("o1", "", "db options (eg. sslmode=disable)")

	var dbUser2 = flag.String("U2", "", "db user")
	var dbPass2 = flag.String("pw2", "", "db password")
	var dbHost2 = flag.String("h2", "localhost", "db host")
	var dbPort2 = flag.Int("p2", 5432, "db port")
	var dbName2 = flag.String("d2", "", "db name")
	var dbOptions2 = flag.String("o2", "", "db options (eg. sslmode=disable)")

	flag.Parse()

	dbInfo1 := pgutil.DbInfo{DbName: *dbName1, DbHost: *dbHost1, DbPort: int32(*dbPort1), DbUser: *dbUser1, DbPass: *dbPass1, DbOptions: *dbOptions1}

	dbInfo2 := pgutil.DbInfo{DbName: *dbName2, DbHost: *dbHost2, DbPort: int32(*dbPort2), DbUser: *dbUser2, DbPass: *dbPass2, DbOptions: *dbOptions2}

	return dbInfo1, dbInfo2
}
