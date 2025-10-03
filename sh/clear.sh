#!/bin/bash

CONSUMER_CONFIG="$HOME/.maanydexd/data"

rm -rf "$CONSUMER_CONFIG/application.db"
rm -rf "$CONSUMER_CONFIG/blockstore.db"
rm -rf "$CONSUMER_CONFIG/cs.wal"
rm -rf "$CONSUMER_CONFIG/evidence.db"
rm -rf "$CONSUMER_CONFIG/snapshots"
rm -rf "$CONSUMER_CONFIG/state.db"
rm -rf "$CONSUMER_CONFIG/tx_index.db"

file="priv_validator_state.json"
tmp_file=$(mktemp)

jq 'del(.signature, .signbytes)
    | .height = "0"
    | .round = 0
    | .step = 0' "$CONSUMER_CONFIG/$file" > "$tmp_file" && mv "$tmp_file" "$CONSUMER_CONFIG/$file"
