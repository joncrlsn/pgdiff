## OSX / Mac / ARM64 pgdiff instructions

These instructions will guide you through the process of generating SQL, reviewing it, and optionally running it on the target database.  It requires a familiarity with the bash shell in OSX.

1. download pgdiff-arm64-\<version\>.tar.gz to your machine
2. untar it (a new directory will be created: called pgdiff)
3. cd into the new pgdiff directory
4. run ./pgdiff ALL --src-url=... --dst-url=...

## tar contents
* pgdiff - an OSX executable
