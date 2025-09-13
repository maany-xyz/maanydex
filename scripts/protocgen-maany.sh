#!/usr/bin/env bash
set -eo pipefail

echo "Generating Maany protos (within root workspace)"
cd proto

# Generate only things under proto/maany/
buf generate --template buf.gen.gogo.yml --path maany

cd ..
# If you rely on go_package-based output into github.com/...:
cp -r github.com/maany-xyz/maany-dex/v5/x/* x/ 2>/dev/null || true
rm -rf github.com
