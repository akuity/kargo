---
sidebar_label: tar
description: Archives a directory or file into a tar (or gzipped tar) file at a specified location.
---

# `tar`

`tar` archives a file or directory into a tar file at a specified location. It supports compressing the archive using gzip.

Ownership is never recorded: every entry in the archive has user and group ID
`0` and no user or group name.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inPath` | `string` | Y | InPath is the path to the source directory or file to be archived. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | OutPath is the path to the destination tar file to create. If the file already exists, it will be overwritten. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `ignore` | `string` | N | Ignore is a (multiline) string of glob patterns to ignore when adding files to the archive. It accepts the same syntax as `.gitignore` files. |
| `gzip` | `boolean` | N | Gzip determines whether the archive should be compressed using gzip. Defaults to false (the archive is uncompressed unless this is set to true). |
| `mtime` | `string` | N | Mtime is an RFC 3339 timestamp (e.g. `"1970-01-01T00:00:00Z"`) to record as the modification time of every entry in the archive, like GNU tar's `--mtime`. Setting it makes the archive depend only on the names, permissions and content of the files, so identical input produces an identical archive. When not set, the files' own modification times are recorded. |

## Examples

### Basic Usage

Archive a directory to a gzipped tar archive:

```yaml
steps:
- uses: tar
  config:
    inPath: ./source-code
    outPath: ./artifacts/bundle.tar.gz
    gzip: true
```

### Uncompressed Archive

Create a standard tar archive without gzip compression:

```yaml
steps:
- uses: tar
  config:
    inPath: ./source-code
    outPath: ./artifacts/bundle.tar
    gzip: false
```

### Ignore Patterns

Archive a directory while ignoring specific files:

```yaml
steps:
- uses: tar
  config:
    inPath: ./source-code
    outPath: ./artifacts/bundle.tar.gz
    gzip: true
    ignore: |
      temp/
      .DS_Store
```

### Fixed Modification Time

By default, each entry in the archive records the modification time of its
file. Files written during a promotion, such as manifests rendered by an
earlier step, get a new modification time every time, so archiving the same
content twice produces two different archives.

Setting `mtime` records the same modification time for every entry instead. The
archive then depends only on the names, permissions and content of the files,
so identical content always produces an identical archive. This is useful when
the archive is pushed with [`oci-push`](oci-push.md): unchanged content keeps
the same digest, and a Warehouse subscribed to that repository creates no new
Freight for it.

```yaml
steps:
- uses: tar
  config:
    inPath: ./rendered
    outPath: ./manifests.tar.gz
    gzip: true
    mtime: "1970-01-01T00:00:00Z"
```
