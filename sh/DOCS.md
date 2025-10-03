# DEX COMMANDS

maanydexd tx gamm create-pool --pool-file ./sh/jsons/pool.json --from alm --gas auto --fees 550000stake --keyring-backend test

maanydexd tx bank send neutron1av4ut027ly6twevug69q62m097mc83yrp28ea5 neutron1a3kufytn7a50jx9hpprh5d2ckzlq6nj2ya7pcz 12999stake --keyring-backend test --gas auto --fees 5500stake

APP_HASH_B64=$(curl -s "$NODE/block?height=7" | jq -r '.result.block.header.app_hash')
APP_HASH_B64=$(echo "$APP_HASH_HEX" | xxd -r -p | base64 | tr -d '\n')
echo "$APP_HASH_B64"
