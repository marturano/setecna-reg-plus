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

## Features

- **One Home Assistant device per zone** (plus the main *Setecna REG* device that holds globals, ACS, circuits, sources, heat pumps and the cascade controller), so a whole zone can be renamed from its device page (requires Home Assistant **2024.11+**, verified up to **2026.7**).
- **Master controls** (when writable): plant **on/off**, **season** (winter/summer) and **ACS on/off**.
- **Native climate entities** (optional *Advanced integration* mode) for each active zone, with heating/cooling `hvac_action`, a single target temperature (the season's comfort setpoint), translated presets (`eco`/`comfort`) and, where available, humidity control.
- **Extra device families** exposed as read-only diagnostics (disabled by default to keep pages clean): heat-pump units and cascade controller, OpenTherm generator cascade (when enabled), board relay outputs, system alarms, zone dew point, circuit return temperatures and pumps, source temperatures, and energy-meter 32-bit totals. Unavailable channels stay *unknown* instead of showing garbage values.
- **Entity renaming from the add-on settings**: name a zone once and all its entities follow.
- **Availability tracking** via MQTT Last Will: entities go `unavailable` if the add-on stops.
- **Self-healing**: automatic re-login when the Setecna session expires, MQTT auto-reconnect with backoff, discovery re-published when Home Assistant restarts, climate entities rebuilt automatically on winter/summer switch.
- **Custom MQTT broker** support, or automatic discovery of the Mosquitto add-on.
- **Self-update entity** that reports the running version and highlights new GitHub releases.

## Installation

1. Make sure you run Home Assistant **2024.11 or newer** and have an MQTT broker (e.g. the Mosquitto broker add-on) plus the MQTT integration configured.
2. Click the button above (or add `https://github.com/marturano/setecna-reg-plus` under *Settings → Add-ons → Add-on store → ⋮ → Repositories*).
3. Install **Setecna REG PLUS**, fill in the configuration and start it.

## Configuration

| Option | Required | Description |
|---|---|---|
| `systemID` | yes | Your system ID, shown in the s5a.eu web interface once logged in |
| `username` | yes | Your s5a.eu account email |
| `password` | yes | Your s5a.eu account password |
| `readonly` | no (`false`) | Only expose sensors; never write to the system |
| `adv_int` | no (`false`) | Create native `climate` entities per zone (requires `readonly: false`) |
| `cleanup_legacy` | no (`true`) | Remove per-entity discovery topics created by add-on v1.x |
| `poll_interval` | no (`30`) | Seconds between refreshes from the Setecna cloud (10–600) |
| `mqtt_host` | no | Custom MQTT broker host. Empty = auto-discover the Mosquitto add-on |
| `mqtt_port` | no (`1883`) | Custom broker port (only with `mqtt_host`) |
| `mqtt_username` | no | Custom broker username (empty = anonymous) |
| `mqtt_password` | no | Custom broker password |
| `entity_names` | no | Friendly-name overrides, one `PREFIX=Name` per entry (see below) |
| `active_zones` | no | Zone numbers to expose; empty = all detected zones |

### Renaming entities

Add one `PREFIX=Name` per entry under `entity_names`:

```
Z1=Bathrooms
Z3=Living room
C1=Panel mixing circuit
GLOBAL_OUTPUT_3=Recirculation pump
```

A **zone prefix** (`Z1`, `Z2`, ...) renames that zone's device and, with it, every entity of the zone and its thermostat at once; an **exact parameter id** (e.g. `GLOBAL_OUTPUT_3`) renames a single entity on the main device. Prefixes and ids are **case-sensitive** (use `Z1`, not `z1`). On startup the add-on logs the custom labels stored in your Setecna panel to copy from.

See [`setecna/DOCS.md`](setecna/DOCS.md) for MQTT topics, diagnostic entities, energy-meter totals, upgrade notes and resilience details.

## Migrating from the original add-on

If you previously ran the original `homeassistant-addon-setecna` (1.x) by ingordigia, entity `unique_id`s are unchanged, so entities, history and dashboards are preserved. Raw state topics moved to `setecna/<systemID>/<PARAM>`; with `cleanup_legacy: true` the old per-entity discovery topics are removed on first start. If you never used the original add-on, just install and go. Full details in the [changelog](setecna/CHANGELOG.md).

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
