//
// Copyright (c) 2014 Jon Carlson.  All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
//

package main

import (
	"github.com/joncrlsn/pgutil"
	flag "github.com/ogier/pflag"
)

func parseFlags() (pgutil.DbInfo, pgutil.DbInfo) {

	var dbUser1 = flag.StringP("user1", "U", "", "db user")
	var dbPass1 = flag.StringP("password1", "W", "", "db password")
	var dbHost1 = flag.StringP("host1", "H", "localhost", "db host")
	var dbPort1 = flag.IntP("port1", "P", 5432, "db port")
	var dbName1 = flag.StringP("dbname1", "D", "", "db name")
	var dbSchema1 = flag.StringP("schema1", "S", "*", "schema name or * for all schemas")
	var dbOptions1 = flag.StringP("options1", "O", "", "db options (eg. sslmode=disable)")

	var dbUser2 = flag.StringP("user2", "u", "", "db user")
	var dbPass2 = flag.StringP("password2", "w", "", "db password")
	var dbHost2 = flag.StringP("host2", "h", "localhost", "db host")
	var dbPort2 = flag.IntP("port2", "p", 5432, "db port")
	var dbName2 = flag.StringP("dbname2", "d", "", "db name")
	var dbSchema2 = flag.StringP("schema2", "s", "*", "schema name or * for all schemas")
	var dbOptions2 = flag.StringP("options2", "o", "", "db options (eg. sslmode=disable)")

	flag.Parse()

	dbInfo1 := pgutil.DbInfo{DbName: *dbName1, DbHost: *dbHost1, DbPort: int32(*dbPort1), DbUser: *dbUser1, DbPass: *dbPass1, DbSchema: *dbSchema1, DbOptions: *dbOptions1}

	dbInfo2 := pgutil.DbInfo{DbName: *dbName2, DbHost: *dbHost2, DbPort: int32(*dbPort2), DbUser: *dbUser2, DbPass: *dbPass2, DbSchema: *dbSchema2, DbOptions: *dbOptions2}

	return dbInfo1, dbInfo2
}
