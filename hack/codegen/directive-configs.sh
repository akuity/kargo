#!/usr/bin/env bash

set -euxo pipefail

out_file=internal/directives/zz_config_types.go
generated_code_warning="// Code generated by quicktype. DO NOT EDIT.\n\n"

rm -rf ${out_file}

quicktype \
  --src-lang schema --alphabetize-properties \
  --lang go --just-types-and-package --package directives --omit-empty \
  -o internal/directives/zz_config_types.go \
  internal/directives/schemas/*.json

printf "${generated_code_warning}$(cat ${out_file})" > ${out_file}

# Pointers to bools and strings don't make a lot of sense in most cases.
#
# Note that -i works on Linux, but not on macOS. -i '' works on macOS, but not
# on Linux. So we use -i.bak, which works on both.
sed -i.bak 's/\*bool/bool/g' ${out_file}
sed -i.bak 's/\*string/string/g' ${out_file}
# As of right now, this transformation is ok, but we can revisit it if we ever
# need nullable numbers or non-int numbers.
sed -i.bak 's/\*float64/int64/g' ${out_file}

rm ${out_file}.bak

gofmt -w ${out_file}
