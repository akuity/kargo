---
sidebar_label: tar
description: Archives a directory or file into a tar (or gzipped tar) file at a specified location.
---

# `tar`

`tar` archives a file or directory into a tar file at a specified location. It supports compressing the archive using gzip.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inPath` | `string` | Y | InPath is the path to the source directory or file to be archived. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | OutPath is the path to the destination tar file to create. If the file already exists, it will be overwritten. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `ignore` | `string` | N | Ignore is a (multiline) string of glob patterns to ignore when adding files to the archive. It accepts the same syntax as `.gitignore` files. |
| `gzip` | `boolean` | N | Gzip determines whether the archive should be compressed using gzip. Defaults to true. |

## Examples

### Basic Usage

Archive a directory to a gzipped tar archive:

```yaml
steps:
- uses: tar
  config:
    inPath: ./source-code
    outPath: ./artifacts/bundle.tar.gz
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
    ignore: |
      temp/
      .DS_Store
```
