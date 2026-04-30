//
// Copyright (c) 2014 Jon Carlson.  All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
//

package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joncrlsn/pgutil"
	flag "github.com/ogier/pflag"
)

func parseFlags() (pgutil.DbInfo, pgutil.DbInfo) {
	var dbUrl1 = flag.String("src-url", "", "source: db connection url (eg. postgres://user:pass@host:port/dbname?sslmode=disable)")
	var dbUser1 = flag.String("src-user", "", "source: db user")
	var dbPass1 = flag.String("src-pass", "", "source: db password")
	var dbHost1 = flag.String("src-host", "localhost", "source: db host")
	var dbPort1 = flag.Int("src-port", 5432, "source: db port")
	var dbName1 = flag.String("src-name", "", "source: db name")
	var dbSchema1 = flag.String("src-schema", "*", "source: schema name or * for all schemas")
	var dbOptions1 = flag.String("src-options", "", "source: db options (eg. sslmode=disable)")

	var dbUrl2 = flag.String("dst-url", "", "destination: db connection url (eg. postgres://user:pass@host:port/dbname?sslmode=disable)")
	var dbUser2 = flag.String("dst-user", "", "destination: db user")
	var dbPass2 = flag.String("dst-pass", "", "destination: db password")
	var dbHost2 = flag.String("dst-host", "localhost", "destination: db host")
	var dbPort2 = flag.Int("dst-port", 5432, "destination: db port")
	var dbName2 = flag.String("dst-name", "", "destination: db name")
	var dbSchema2 = flag.String("dst-schema", "*", "destination: schema name or * for all schemas")
	var dbOptions2 = flag.String("dst-options", "", "destination: db options (eg. sslmode=disable)")

	var excludeCheckFlag = flag.String("exclude-check", "", "comma-separated check constraints to skip, format: schema.table.constraint")

	flag.Parse()

	set, err := parseExcludeList(*excludeCheckFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --exclude-check: %v\n", err)
		os.Exit(1)
	}
	excludeChecks = set

	dbInfo1 := buildDbInfo(*dbUrl1, *dbUser1, *dbPass1, *dbHost1, *dbPort1, *dbName1, *dbSchema1, *dbOptions1)
	dbInfo2 := buildDbInfo(*dbUrl2, *dbUser2, *dbPass2, *dbHost2, *dbPort2, *dbName2, *dbSchema2, *dbOptions2)

	return dbInfo1, dbInfo2
}

// parseExcludeList splits a comma-separated list of schema.table.constraint
// patterns into a set. validates every token up-front; rejects empty parts so
// we don't silently accept trailing dots.
func parseExcludeList(raw string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if strings.TrimSpace(raw) == "" {
		return out, nil
	}
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		parts := strings.Split(t, ".")
		if len(parts) != 3 {
			return nil, fmt.Errorf("expected schema.table.constraint, got %q", t)
		}
		for _, p := range parts {
			if strings.TrimSpace(p) == "" {
				return nil, fmt.Errorf("expected schema.table.constraint, got %q", t)
			}
		}
		out[t] = struct{}{}
	}
	return out, nil
}

// buildDbInfo prefers the connection url when set; otherwise falls back to individual flags.
func buildDbInfo(dbUrl, user, pass, host string, port int, name, schema, options string) pgutil.DbInfo {
	if dbUrl == "" {
		return pgutil.DbInfo{
			DbName:    name,
			DbHost:    host,
			DbPort:    int32(port),
			DbUser:    user,
			DbPass:    pass,
			DbSchema:  schema,
			DbOptions: options,
		}
	}

	info, err := dbInfoFromUrl(dbUrl, schema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid url: %v\n", err)
		os.Exit(1)
	}
	return info
}

// dbInfoFromUrl parses a postgres connection url into a DbInfo.
// schema query param overrides the schema flag fallback when present.
func dbInfoFromUrl(raw, schemaFallback string) (pgutil.DbInfo, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return pgutil.DbInfo{}, err
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return pgutil.DbInfo{}, fmt.Errorf("unsupported scheme %q (expected postgres:// or postgresql://)", u.Scheme)
	}

	port := 5432
	if p := u.Port(); p != "" {
		port, err = strconv.Atoi(p)
		if err != nil {
			return pgutil.DbInfo{}, fmt.Errorf("invalid port %q: %w", p, err)
		}
	}

	var user, pass string
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}

	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}

	// schema is not part of the standard postgres url; accept it via a query param,
	// otherwise fall back to whatever was passed via --src-schema / --dst-schema.
	q := u.Query()
	schema := schemaFallback
	if s := q.Get("schema"); s != "" {
		schema = s
		q.Del("schema")
	}

	// remaining query params become space-separated key=value options for libpq.
	var optParts []string
	for k, vals := range q {
		for _, v := range vals {
			optParts = append(optParts, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return pgutil.DbInfo{
		DbName:    strings.TrimPrefix(u.Path, "/"),
		DbHost:    host,
		DbPort:    int32(port),
		DbUser:    user,
		DbPass:    pass,
		DbSchema:  schema,
		DbOptions: strings.Join(optParts, " "),
	}, nil
}
