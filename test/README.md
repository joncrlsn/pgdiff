These are not automated tests (I'd rather have automated tests), but manual
integration tests for verifying that individual schema types are working.

These can be good templates for isolating bugs in the different data diffs.

Connect to the database manually:
  sudo su - postgres -- -c "psql -d db1"

