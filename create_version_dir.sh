#!/usr/bin/env bash

set -e

version="${1:?"Usage: create_version_dir.sh <version>"}"

if [[ ! "$version" =~ ^v ]]; then
    version="v$version"
fi

go_files=$(find . ! -path "*/vendor/*" ! -path "*/fakes/*" ! -path "*/tools/*" ! -path "*/v[0-9]*/*" ! -name "*_test.go" -name "*.go")
for f in $go_files ; do
    mkdir -p "$version/$(dirname $f)"
    cp $f $version/$(dirname $f)
done
