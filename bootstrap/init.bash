#!/usr/bin/env bash

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <module-name>"
  exit 1
fi

MODULE_NAME=$1

grep -rl your-module-name | xargs sed -i "s/your-module-name/$MODULE_NAME/g"
rm -rf bootstrap
rm gitsync.json

echo -e "# $MODULE_NAME\nTODO\n" > README.md
