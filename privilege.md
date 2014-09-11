SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'information_schema';

SELECT grantee, table_name, privilege_type, is_grantable, with_hierarchy
FROM information_schema.table_privileges
WHERE table_schema = 'public'
ORDER BY table_name, privilege_type;

SELECT grantee, table_name, column_name, privilege_type, is_grantable 
FROM information_schema.column_privileges 
WHERE table_schema = 'public';
