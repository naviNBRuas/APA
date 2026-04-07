# Harnesses for Integration & System Testing

This directory contains docker-compose and related harnesses for integration, system, and scenario testing of the APA agent and its components.

## Structure
- Each subfolder (e.g., `p2p/`) is a self-contained harness for a specific scenario or integration test.
- Harnesses typically include a `docker-compose.yml`, a `README.md` with usage instructions, and any shared runtime files or scripts.

## Adding a New Harness
1. Copy the `harness-template/` folder and rename it for your scenario.
2. Edit the `docker-compose.yml` and `README.md` as needed.
3. Add any scripts or shared folders required for your test.

## Existing Harnesses
- `p2p/`: P2P relay/circuit integration harness.

---

**Note:** Harnesses are for development, research, and academic demonstration only. See project disclaimer.
