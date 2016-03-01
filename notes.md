 ==== Constraints

SELECT c.conname AS constraint_name,
          CASE c.contype
            WHEN 'c' THEN 'CHECK'
            WHEN 'f' THEN 'FOREIGN KEY'
            WHEN 'p' THEN 'PRIMARY KEY'
            WHEN 'u' THEN 'UNIQUE'
          END AS "constraint_type",
          CASE WHEN c.condeferrable = 'f' THEN 0 ELSE 1 END AS is_deferrable,
          CASE WHEN c.condeferred = 'f' THEN 0 ELSE 1 END AS is_deferred,
          t.relname AS table_name,
          array_to_string(c.conkey, ' ') AS constraint_key,
          CASE confupdtype
            WHEN 'a' THEN 'NO ACTION'
            WHEN 'r' THEN 'RESTRICT'
            WHEN 'c' THEN 'CASCADE'
            WHEN 'n' THEN 'SET NULL'
            WHEN 'd' THEN 'SET DEFAULT'
          END AS on_update,
          CASE confdeltype
            WHEN 'a' THEN 'NO ACTION'
            WHEN 'r' THEN 'RESTRICT'
            WHEN 'c' THEN 'CASCADE'
            WHEN 'n' THEN 'SET NULL'
            WHEN 'd' THEN 'SET DEFAULT'
          END AS on_delete,
          CASE confmatchtype
            WHEN 'u' THEN 'UNSPECIFIED'
            WHEN 'f' THEN 'FULL'
            WHEN 'p' THEN 'PARTIAL'
          END AS match_type,
          t2.relname AS references_table,
          array_to_string(c.confkey, ' ') AS fk_constraint_key
     FROM pg_constraint c
LEFT JOIN pg_class t  ON c.conrelid  = t.oid
LEFT JOIN pg_class t2 ON c.confrelid = t2.oid;


   SELECT tc.constraint_name
          , tc.constraint_type
          , tc.table_name
          , kcu.column_name
          , tc.is_deferrable
          , tc.initially_deferred
          , rc.match_option AS match_type
          , rc.update_rule AS on_update
          , rc.delete_rule AS on_delete
          , ccu.table_name AS references_table
          , ccu.column_name AS references_field
     FROM information_schema.table_constraints tc
LEFT JOIN information_schema.key_column_usage kcu
       ON tc.constraint_catalog = kcu.constraint_catalog
      AND tc.constraint_schema = kcu.constraint_schema
      AND tc.constraint_name = kcu.constraint_name
LEFT JOIN information_schema.referential_constraints rc
       ON tc.constraint_catalog = rc.constraint_catalog
      AND tc.constraint_schema = rc.constraint_schema
      AND tc.constraint_name = rc.constraint_name
LEFT JOIN information_schema.constraint_column_usage ccu
       ON rc.unique_constraint_catalog = ccu.constraint_catalog
      AND rc.unique_constraint_schema = ccu.constraint_schema
      AND rc.unique_constraint_name = ccu.constraint_name
WHERE tc.constraint_schema = 'public'
AND tc.constraint_type = 'UNIQUE';


 ==== Trigger and Function

SELECT *
  FROM information_schema.triggers
   WHERE trigger_schema NOT IN
          ('pg_catalog', 'information_schema');


CREATE TABLE emp (
    empname text,
    salary integer,
    last_date timestamp,
    last_user text
);

CREATE FUNCTION emp_stamp() RETURNS trigger AS $emp_stamp$
    BEGIN
        -- Check that empname and salary are given
        IF NEW.empname IS NULL THEN
            RAISE EXCEPTION 'empname cannot be null';
        END IF;
        IF NEW.salary IS NULL THEN
            RAISE EXCEPTION '% cannot have null salary', NEW.empname;
        END IF;

        -- Who works for us when she must pay for it?
        IF NEW.salary < 0 THEN
            RAISE EXCEPTION '% cannot have a negative salary', NEW.empname;
        END IF;

        -- Remember who changed the payroll when
        NEW.last_date := current_timestamp;
        NEW.last_user := current_user;
        RETURN NEW;
    END;
$emp_stamp$ LANGUAGE plpgsql;

CREATE TRIGGER emp_stamp BEFORE INSERT OR UPDATE ON emp
    FOR EACH ROW EXECUTE PROCEDURE emp_stamp();

CREATE TRIGGER check_update
    BEFORE UPDATE ON emp
    FOR EACH ROW
    WHEN (OLD.empname IS DISTINCT FROM NEW.empname)
    EXECUTE PROCEDURE emp_stamp();
