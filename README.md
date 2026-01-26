# Weaver

Weaver combines files from a directory into a single text file. It uses `.gitignore`-compatible
pattern syntax to filter files and lets you apply multiple blacklist/whitelist rule files in order.
You can optionally embed a JSON directory tree of included files in the output.

## Features

- `.gitignore` pattern syntax (globs, `**`, negation, anchored rules)
- multiple blacklist/whitelist rule files with ordered precedence
- optional JSON tree of included files in the combined output
- deterministic output ordering

## Usage

```bash
# Use current directory as root. Include all files. Print to stdout.
weaver

# See help and available flags.
weaver --help

weaver -root . -out combined.txt
weaver -root . -out combined.txt -whitelist allowlist.txt
weaver -root . -out - -include-tree
weaver -root ./api -root ./web -out -
weaver -blacklist .gitignore -whitelist .allowed -out combined.txt
```

### Flags

- `-root`: root directory to scan (repeatable, defaults to the current directory)
- `-out`: output file path (`-` for stdout, defaults to stdout)
- `-blacklist`: path to a gitignore-style file to blacklist (repeatable)
- `-whitelist`: path to a gitignore-style file to whitelist (repeatable)
- `-include-tree`: include JSON file tree in output

## Notes

- Rule files are evaluated in the order provided; later matches override earlier ones.
- If no rule files are provided, all files are included.
- In whitelist rules, directory-only patterns (ending in `/`) include all files under that directory.
- The output file is automatically excluded if it lives under a root directory.

## Build

```bash
go build ./cmd/weaver
```

## Test

```bash
go test ./...
```
