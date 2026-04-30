## Linux arm64 pgdiff instructions

These instructions will guide you through the process of generating SQL, reviewing it, and optionally running it on the target database. It requires a familiarity with a Linux command-line shell. This build targets aarch64/arm64 hosts (e.g. AWS Graviton / Fargate ARM tasks).

1. download pgdiff-linux-arm64-\<version\>.tar.gz to your machine
2. untar it (a new directory will be created: called pgdiff)
3. cd into the new pgdiff directory
4. run ./pgdiff ALL --src-url=... --dst-url=...

## tar contents
* pgdiff - a linux arm64 executable
