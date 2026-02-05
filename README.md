# Weaver

Weaver combines files from a directory into a single text file. It uses `.gitignore`-compatible
pattern syntax to filter files and lets you apply multiple blacklist/whitelist rule files in order.
You can optionally embed a JSON directory tree of included files in the output.

## Features

- `.gitignore` pattern syntax (globs, `**`, negation, anchored rules)
- multiple blacklist/whitelist rule files with ordered precedence
- inline blacklist/whitelist patterns via CLI flags
- optional JSON tree of included files in the combined output
- optional max depth for directory walking
- optional skipping of file contents or binary payloads
- deterministic output ordering

## Usage

```bash
# Use current directory as root. Include all files. Print to stdout.
weaver

# See help and available flags.
weaver --help

weaver -root . -out combined.txt
weaver -root . -out combined.txt -whitelist allowlist.txt
weaver -root . -out - -blacklist-pattern "*.log"
weaver -root . -out - -include-tree
weaver -root . -out - -include-tree-compact
weaver -root . -out - -max-depth 2 -skip-binary
weaver -root ./api -root ./web -out -
weaver -blacklist .gitignore -whitelist .allowed -out combined.txt
```

### Flags

- `-root`: root directory to scan (repeatable, defaults to the current directory)
- `-out`: output file path (`-` for stdout, defaults to stdout)
- `-blacklist`: path to a gitignore-style file to blacklist (repeatable)
- `-whitelist`: path to a gitignore-style file to whitelist (repeatable)
- `-blacklist-pattern`: inline gitignore-style blacklist pattern (repeatable)
- `-whitelist-pattern`: inline gitignore-style whitelist pattern (repeatable)
- `-include-tree`: include JSON file tree in output
- `-include-tree-compact`: include JSON file tree as a one-line payload
- `-max-depth`: max directory depth to include (`-1` for no limit, `0` for root only)
- `-skip-contents`: skip writing file contents (header and optional tree only)
- `-skip-binary`: replace binary file contents with a placeholder line

## Notes

- Rule files are evaluated in the order provided; later matches override earlier ones.
- If no rule files are provided, all files are included.
- In whitelist rules, directory-only patterns (ending in `/`) include all files under that directory.
- The output file is automatically excluded if it lives under a root directory.
- Use `-include-tree` and `-include-tree-compact` together to include both tree formats.
- Binary detection uses a lightweight heuristic (NUL bytes or a high ratio of control characters) and is best-effort.

## Build

```bash
go build ./cmd/weaver
```

## Test

```bash
go test ./...
```
