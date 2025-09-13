#!/usr/bin/env bash
set -eo pipefail

cd proto

echo "Generating gogo proto code for Osmosis"
# Generate ONLY Osmosis files with Osmosis' own buf config (SDK 0.47)
while IFS= read -r -d '' file; do
  if grep -q "option go_package" "$file"; then
    echo "  osmo: $file"
    buf generate "$file" \
      --config ./osmosis/buf.osmo.yaml \
      --template ./osmosis/buf.gen.gogo.yaml
  fi
done < <(find ./osmosis -name '*.proto' -print0)

echo "Generating gogo proto code for Neutron fork (SDK 0.50)"
# Everything else uses your root buf.yaml (SDK 0.50)
while IFS= read -r -d '' file; do
  if grep -q "option go_package" "$file"; then
    echo "  neutron: $file"
    buf generate --template buf.gen.gogo.yml "$file"
  fi
done < <(find . -path ./osmosis -prune -o -name '*.proto' -print0)

cd ..

# Copy generated code from go_package paths to your x/ tree (minimal)
# Keep this list tight; copy only what you need:
cp -r github.com/osmosis-labs/osmosis/osmoutils ./osmoutils || true
cp -r github.com/osmosis-labs/osmosis/x/epochs ./x/ || true
# If you use GAMM or others, add them explicitly:
# cp -r github.com/osmosis-labs/osmosis/x/gamm ./x/ || true

# Your existing Neutron/DEX copy step (if you still generate under go_package paths)
# cp -r github.com/maany-xyz/maany-dex/v5/x/* x/ || true

rm -rf github.com
