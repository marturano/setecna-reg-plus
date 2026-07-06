# DISCLAIMER

This add-on is developed by reverse engineering the Setecna web-interface and is not officially supported by the Setecna team, use it with caution.

Setecna REG PLUS is a fork of the original [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) by **ingordigia**, rewritten in Go and substantially extended. Distributed under the Apache-2.0 license.

# Home Assistant Add-on: Setecna REG PLUS

> This documentation is in English. A translated overview is available in the repository README: [Italiano](../README.it.md) · [Deutsch](../README.de.md) · [Français](../README.fr.md) · [Español](../README.es.md). The add-on configuration UI is localised in English, Italian, German, French and Spanish.

## Why this add-on

This add-on integrates your Setecna REG based thermal plant into Home Assistant. It is a web based integration: your system needs internet access to communicate with the Setecna servers (s5a.eu), and the add-on bridges that data to Home Assistant over MQTT.

The add-on uses **MQTT device-based discovery** (requires Home Assistant 2024.11 or newer): the whole REG system appears as a single MQTT device with all of its sensors, controls and (optionally) native climate entities.

## Prerequisites

1. Home Assistant 2024.11 or newer (verified up to 2026.7: the discovery payload uses no options deprecated or removed through 2026.7, such as the removed `object_id`)
2. An MQTT broker add-on (e.g. Mosquitto broker) installed and started
3. The MQTT integration enabled and configured

*By default the add-on automatically discovers the broker host, port and credentials through the Supervisor services API. Alternatively, you can point it to any external broker with the `mqtt_host` / `mqtt_port` / `mqtt_username` / `mqtt_password` options.*

## Configuration

| Option | Required | Description |
|---|---|---|
| `systemID` | yes | Your system ID, visible in the s5a.eu web interface once logged in |
| `username` | yes | Your s5a.eu account email |
| `password` | yes | Your s5a.eu account password |
| `readonly` | no (default `false`) | Only expose sensors; never write anything back to the system |
| `adv_int` | no (default `false`) | Create native `climate` entities for each active zone. Requires `readonly: false` |
| `cleanup_legacy` | no (default `true`) | On startup, publish removal messages for the per-entity discovery topics used by add-on v1.x |
| `poll_interval` | no (default `30`) | Seconds between refreshes from the Setecna cloud (10-600) |
| `mqtt_host` | no | Hostname/IP of a custom MQTT broker. Leave empty to auto-discover the Mosquitto broker add-on via the Supervisor |
| `mqtt_port` | no (default `1883`) | Port of the custom MQTT broker, only used together with `mqtt_host` |
| `mqtt_username` | no | Username for the custom broker (empty = anonymous) |
| `mqtt_password` | no | Password for the custom broker |
| `entity_names` | no | Friendly-name overrides, one `PREFIX=Name` per entry (see *Renaming entities* below) |

## MQTT topics

- Discovery (retained): `homeassistant/device/setecna_<systemID>/config`
- Availability (retained, with Last Will): `setecna/<systemID>/availability`
- States (retained): `setecna/<systemID>/<PARAM>`
- Commands (from Home Assistant): `setecna/<systemID>/<PARAM>/set`

## Migrating from the original add-on

If you previously ran the original `homeassistant-addon-setecna` (1.x) by ingordigia, entity `unique_id`s are unchanged, so your entities, history and dashboards are preserved. What changes:

- Raw state topics moved from the old `homeassistant/<type>/<systemID>_<PARAM>` layout to `setecna/<systemID>/<PARAM>`. If you had automations reading the raw MQTT topics (instead of the HA entities), update them.
- On first start, with `cleanup_legacy: true`, the add-on removes the old per-entity discovery topics left by the original add-on. If some stale entities remain, restart Home Assistant once. If you never used the original add-on, this option is harmless and can be left on.

## Resilience

- The add-on automatically re-authenticates when the s5a.eu session expires.
- MQTT reconnects automatically; while the add-on is down all entities are marked `unavailable` via the Last Will message.
- Discovery and states are re-published whenever Home Assistant restarts (birth message) or the broker reconnects.
- When the system switches between winter and summer, climate entities are rebuilt automatically with the correct seasonal setpoints.
- A self-update entity reports the running version and highlights newer GitHub releases.

## Renaming entities

By default entities are named generically ("Zone 1 temperature", "Circuit 1 return temperature", ...). You can rename them from the add-on configuration with the **entity name overrides** option, one `PREFIX=Name` per entry:

```
Z1=Bagni
Z3=Soggiorno
C1=Panel mixing circuit
GLOBAL_OUTPUT_3=Recirculation pump
```

- A zone/circuit/source/heat-pump **prefix** (`Z1`, `C1`, `S1`, `HP0`) renames every entity of that element and its thermostat in one go: `Z1=Bagni` produces "Bagni temperature", "Bagni dew point", the "Bagni" thermostat, etc.
- An **exact parameter id** (e.g. `GLOBAL_OUTPUT_3`) renames a single entity.

To discover which zone is which, the add-on prints, on every start, the custom labels you configured on the Setecna panel (`_FREEDESC`/`_XFREEDESC`) and the description code of each active zone. Look at the add-on log after startup and copy the labels into the option. (Automatic naming from the Setecna description codes is intentionally not done: the built-in description dictionary of the controller is not documented, so guessing could mislabel rooms.)

Entity `unique_id`s never change, so renaming here — or directly in the Home Assistant UI — is preserved across restarts and add-on updates.

## Diagnostic entities (heat pumps, boilers, sources)

Beyond zones and circuits, the add-on exposes the heat-pump units and cascade controller, the OpenTherm generator cascade (only if enabled on your system), the board relay outputs and the system alarms, all as read-only diagnostic entities. Channels that report a "not available" sentinel are shown as *unknown* rather than a fake number.

Some fields (power, status, error and request codes) are exposed as **raw values** because their exact unit or encoding has not been reverse engineered yet; their names end with "(raw)" or "code". If you work out the correct scale on your system, please open an issue.

### Energy meter 32-bit totals

Each energy accumulator is split by the controller into a low word (`ACCLO`/`ACC2LO`, exposed as kWh) and a high word (`ACCHI`/`ACC2HI`). For totals that exceed the 16-bit low word, reconstruct the full value with a Home Assistant template sensor:

```
{{ (states('sensor.energy_meter_1_total_energy_import_high_word') | int * 65536
    + states('sensor.energy_meter_1_total_energy_import') | float * 10) / 10 }}
```

## Climate entities

For each active zone the *Advanced integration* mode creates a `climate` entity:
- The HVAC **mode** follows the zone forcing: `off` when the zone is forced off, `heat`/`cool` otherwise. Changing the mode from the card writes the forcing back to the system (heat/cool -> automatic, off -> forced off).
- The **hvac action** (heating/cooling/idle) reflects the actual zone relay output.
- **Presets** (`forced off` / `forced economy` / `forced comfort`) expose the finer forcing levels.
- The low/high target temperatures map to the seasonal economy/comfort setpoints.
