# Maany DEX

## Modules Docs

- MintBurn (consumer-side privileged ICS-20 mint path bound to CCV): `x/mintburn/README.md`
- GenesisMint (genesis proof-mint + ICA claim confirmation): `x/genesismint/README.md`

# Swap Routing Logic

[swap-exact-amount-in]:
gamm's msg_server.go -> `SwapExactAmountIn` -> poolmanager's router.go `RouteExactAmountIn` which loops through hops and for every hop calls poolmodue's rounter.go `SwapExactAmountIn` (and handles taker fee logic) -> calls gamm implementation of `SwapExactAmountIn` from swap.go
