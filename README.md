![Neutron](https://github.com/neutron-org/neutron-docs/blob/1db1e92098c915ae8ad4defc0bd30ef549175201/static/img/neutron_wide_logo.png)

# Neutron

## Documentation

You can check the documentation here: https://docs.neutron.org/

## Build

You can check out the build instructions here: https://docs.neutron.org/neutron/build-and-run/overview

## Sync and join the network

```bash
bash contrib/statesync.bash
```

## Examples

You can check out the example contracts here: https://github.com/neutron-org/neutron-contracts

## Tests

Integration tests are implemented here: https://github.com/neutron-org/neutron-integration-tests

# Swap Routing Logic

[swap-exact-amount-in]:
gamm's msg_server.go -> `SwapExactAmountIn` -> poolmanager's router.go `RouteExactAmountIn` which loops through hops and for every hop calls poolmodue's rounter.go `SwapExactAmountIn` (and handles taker fee logic) -> calls gamm implementation of `SwapExactAmountIn` from swap.go

# TODO:

2. Change project name (from ../neutrond to actual github repo name)
3. Change BIN name
4. Change denom "umaany"
5. change bench32 name

# and in config

- Change chain-id (other neutron related naming?)
- Change turn off unused neutron modules

# Commands

`neutrond tx ibc-transfer transfer transfer channel-7 maany1evvmj6yr7nv29tk7yp5z3qaru308rt5acw4mqp 333333333umaany --from cons-key --gas auto --gas-adjustment 1.5 --fees 200000umaany --keyring-backend test --chain-id maanydex -y`

`neutrond tx bank send maany-dex1c7j33khjnqf2s44t2aykshthjkpf50sfxzu7af maany-dex19k2p7rdqvjcm7yq57c6ntgsfna857pq79fgmpt 1000000000umaany`
