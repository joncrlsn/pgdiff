SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'information_schema';

SELECT grantee, table_name, privilege_type, is_grantable, with_hierarchy, 
FROM information_schema.table_privileges
WHERE table_schema = 'public'
ORDER BY table_name, privilege_type;

SELECT grantee, table_name, column_name, privilege_type, is_grantable 
FROM information_schema.column_privileges 
WHERE table_schema = 'public';


--
-- GRANT SQL
--
GRANT SELECT (service_type) ON t_computer TO c42ro;

-- selects grants for tables, views, and sequences
SELECT relname
    , relacl
    , CASE WHEN relkind = 'S' THEN 'SEQUENCE' 
        WHEN relkind = 'r' THEN 'TABLE' 
        WHEN relkind = 'v' THEN 'VIEW' 
        ELSE relkind::varchar END AS type
FROM pg_class
WHERE true --relkind = 'S'
  AND relacl IS NOT NULL 
  AND relnamespace IN (
      SELECT oid FROM pg_namespace
      WHERE nspname NOT LIKE 'pg_%' AND nspname != 'information_schema'
);


crashplan=# \dp
********* QUERY **********
SELECT n.nspname as "Schema",
  c.relname as "Name",
  CASE c.relkind WHEN 'r' THEN 'table' WHEN 'v' THEN 'view' WHEN 'S' THEN 'sequence' WHEN 'f' THEN 'foreign table' END as "Type",
  pg_catalog.array_to_string(c.relacl, E'\n') AS "Access privileges",
  pg_catalog.array_to_string(ARRAY(
    SELECT attname || E':\n  ' || pg_catalog.array_to_string(attacl, E'\n  ')
    FROM pg_catalog.pg_attribute a
    WHERE attrelid = c.oid AND NOT attisdropped AND attacl IS NOT NULL
  ), E'\n') AS "Column access privileges"
FROM pg_catalog.pg_class c
     LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r', 'v', 'S', 'f')
  AND n.nspname !~ '^pg_' AND pg_catalog.pg_table_is_visible(c.oid)
ORDER BY 1, 2;
**************************
