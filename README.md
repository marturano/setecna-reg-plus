# DISCLAIMER

This is an unofficial, community-developed integration. It is **not affiliated with, endorsed by, sponsored by, or supported in any way by SETECNA EPC Srl**, the provider of the Setecna REG system and of the s5a.eu (www.s5a.eu) remote-management portal.

"Setecna", "Setecna REG", "s5a.eu" and any related names or logos are trademarks or property of their respective owners, used here solely for identification and interoperability. This project is developed independently, by reverse-engineering the publicly reachable s5a.eu web interface.

**No warranty.** The software is provided "as is", without warranty of any kind, express or implied, including but not limited to merchantability, fitness for a particular purpose, accuracy or non-infringement.

**Limitation of liability.** To the maximum extent permitted by law, the author(s) accept no liability for any damage, loss or cost of any kind - direct, indirect, incidental, special or consequential - arising from the use of, or inability to use, this integration. This includes, without limitation: malfunction, damage or wear of the heating/cooling plant or its components; incorrect, missing or delayed readings and commands; loss of comfort, energy waste or increased costs; data loss; interruption or suspension of the s5a.eu service or of your account.

**Use at your own risk.** The integration can read and, when enabled, **write** parameters to your Setecna REG system (setpoints, season, on/off, forcing, ...). Such writes change the actual behaviour of your plant. You are solely responsible for how you use it and for any change it makes to your system. This integration is **not a safety device** and must not be relied upon for any safety-critical function.

**Third-party terms.** You are responsible for ensuring your use complies with SETECNA EPC Srl's terms of service for the Setecna REG system and the s5a.eu portal. Access may be changed or withdrawn by the provider at any time, which may stop this integration from working.

# Home Assistant Setecna REG add-on repository

> 🌐 **Languages:** 🇬🇧 English · 🇮🇹 [Italiano](README.it.md) · 🇩🇪 [Deutsch](README.de.md) · 🇫🇷 [Français](README.fr.md) · 🇪🇸 [Español](README.es.md)

This repository contains the Home Assistant add-on that integrates a **Setecna REG** thermal system into Home Assistant over MQTT, using [device-based discovery](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery).

[![Open your Home Assistant instance and show the add add-on repository dialog with a specific repository URL pre-filled.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fmarturano%2Fsetecna-reg-plus)

![Supports aarch64 Architecture][aarch64-shield]
![Supports amd64 Architecture][amd64-shield]

---

## How it works

Setecna REG systems are managed through the **s5a.eu** cloud portal. The add-on logs into that portal, polls the full system snapshot at a regular interval and republishes everything to Home Assistant over **MQTT** using device-based discovery; commands from Home Assistant are translated back into portal writes.

```
Setecna REG plant  <->  s5a.eu cloud  <->  [ add-on ]  <->  MQTT broker  <->  Home Assistant
```

The plant appears as a **main device** (system-wide sensors and controls) plus **one sub-device per active zone**. Entity `unique_id`s are stable, so history, dashboards and manual renames survive restarts and updates.

## Features

- **One Home Assistant device per zone**, plus the main *Setecna REG* device (globals, ACS, circuits, sources, energy meters, calendars and optional diagnostics). Requires Home Assistant **2024.11+**.
- **Native thermostat** per zone (optional): single-setpoint `heat`/`cool` + `off`, current temperature, and current/target humidity (0-100%) when a probe is present. Tuned to work reliably with **Amazon Alexa**.
- **Regime** sensor (live automatic/forced x comfort/eco, correct even when idle) and a **Forcing** select to override the schedule.
- **Off is shown when the zone is really off** (forced or by schedule), not only when manually forced.
- **Per-zone calendars**, seasonal comfort/economy setpoints, dew point, zone state.
- **Diagnostic entities** (optional): heat pumps, cascade controller, OpenTherm cascade, relay outputs, system alarms.
- **`hide_unavailable`** (default on): channels not wired/installed on your system are hidden instead of showing as *unknown*.
- **Entity renaming**, **master controls** (system/season/ACS), **zone filtering**, **five languages** (en/it/de/fr/es).
- **Resilient**: cloud re-login, MQTT auto-reconnect with availability (Last Will), re-publish on HA restart, backoff on failures, and a self-update entity.

## Installation

1. Add this repository to the Home Assistant add-on store and install **Setecna REG PLUS**.
2. In the add-on **Configuration**, set at least `systemID`, `username` and `password` (your s5a.eu account).
3. Start the add-on. The Setecna device and its zones appear under **Settings -> Devices & Services -> MQTT**.

You need an MQTT broker (e.g. *Mosquitto broker*) and the MQTT integration enabled. The broker is auto-discovered via the Supervisor, or set manually with the `mqtt_*` options.

## Configuration

The most useful options (see [`setecna/DOCS.md`](setecna/DOCS.md) for the full table and detailed explanations):

| Option | Default | Description |
|---|---|---|
| `systemID` / `username` / `password` | - | **Required** s5a.eu credentials. |
| `readonly` | `false` | Expose sensors only; create no controls. |
| `adv_int` | `false` | Create a thermostat per zone (needs `readonly: false`). |
| `diagnostics` | `false` | Expose heat pumps / OpenTherm / relays / alarms. |
| `hide_unavailable` | `true` | Hide not-available channels instead of showing *unknown*. |
| `language` | `en` | `en` / `it` / `de` / `fr` / `es`. |
| `active_zones` | `[]` | Limit which zones are exposed (empty = all). |
| `entity_names` | `[]` | Rename entities (`Z1=Bagni`, ...). |
| `poll_interval` | `30` | Seconds between cloud refreshes (10-600). |

Full documentation, including the thermostat behaviour, the Regime-vs-Forcing distinction, Alexa notes, renaming and troubleshooting, is in **[`setecna/DOCS.md`](setecna/DOCS.md)**.

## Migrating from the original add-on

Entity `unique_id`s are unchanged, so entities, history and dashboards are preserved. On first start `cleanup_legacy` removes the old per-entity discovery topics; if stale entities remain, restart Home Assistant once.

## Development

The add-on is written in Go (see [`setecna/`](setecna/)). Continuous integration runs `gofmt`, `go vet`, `go build` and `go test -race` on every push (see [`.github/workflows/go.yaml`](.github/workflows/go.yaml)). The helper script [`setecna/tools/setecna_diff.py`](setecna/tools/setecna_diff.py) compares a full `getres` dump against the add-on's coverage and lists unmapped parameters.

```bash
cd setecna
go test ./...
go build ./cmd
```

---

## Credits

This add-on is a fork of the original [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) by **ingordigia**, rewritten in Go and substantially extended. Distributed under the Apache-2.0 license.

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
