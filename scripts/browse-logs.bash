#!/usr/bin/env bash

if [[ $# -ne 1 ]]; then
  echo "Exactly one argument expected." >&2
  echo "Usage: $0 <file-path>" >&2
  exit 1
fi

FILE="$1"

if [[ ! -f "$FILE" ]]; then
  echo "Error: '$FILE' is not a valid file." >&2
  exit 1
fi

tac "$FILE" |
  jq -r '. | "\(.level)\t\(.msg)\t\(. | @json)"' |
  fzf \
    --no-sort \
    --delimiter="\t" \
    --with-nth=1,2 \
    --preview 'echo {3} | jq "." | bat -f --language json --style=plain'
