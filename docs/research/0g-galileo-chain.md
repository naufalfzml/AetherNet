# 0G Galileo Chain Configuration

Last checked: 2026-05-04

## Decision

Use `16601` as the 0G Galileo Testnet chain ID for AetherNet.

## Network Values

| Field | Value |
| --- | --- |
| Network name | `0G-Galileo-Testnet` |
| Chain ID | `16601` |
| RPC URL | `<0G_GALILEO_RPC_URL>` |
| Explorer | `<0G_GALILEO_EXPLORER_URL>` |
| Native token | `OG` |

## Sources Checked

- Chain-List entry for 0G Galileo Testnet lists chain ID `16601`, RPC URL, explorer URL, and native currency `OG`.
- Official 0G Foundation deployment examples repository (`0gfoundation/0g-deployment-scripts`) lists Testnet V3 / Galileo with chain ID `16601`, RPC URL, token symbol `OG`, and explorer URL.

## Conflict Notes

Some older community mirrors and social reposts mention `80087`. For implementation, prefer the 0G Foundation deployment scripts plus Chain-List's current Galileo entry, both of which agree on `16601`.

Recheck this before a live deployment because testnet configuration can change.
