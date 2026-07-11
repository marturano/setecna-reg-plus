# DISCLAIMER

This add-on is developed by reverse engineering the Setecna web interface and is
**not** officially supported by Setecna. Use it at your own risk. It writes to
your heating/cooling system only when you allow it (see `readonly`).

# Setecna REG PLUS

Integrates a **Setecna REG** based thermal plant into Home Assistant.

---

## How it works

Setecna REG systems are managed through the **s5a.eu** cloud portal. This add-on
logs into that portal with your account, polls the full system snapshot at a
regular interval, and republishes everything to Home Assistant over **MQTT**,
using **device-based MQTT discovery**. Commands you send from Home Assistant are
translated back into portal writes.

```
Setecna REG plant  <->  s5a.eu cloud  <->  [ add-on ]  <->  MQTT broker  <->  Home Assistant
```

Because the path is cloud based, your Home Assistant instance and the add-on
need Internet access to s5a.eu; there is no direct local connection to the panel.

The whole plant is exposed as a **main device** (system-wide sensors and
controls) plus one **sub-device per active zone** (its sensors, controls and
optional thermostat). Entity `unique_id`s are stable, so history, dashboards and
manual renames survive restarts and updates.

---

## Prerequisites

1. **Home Assistant 2024.11 or newer** (device-based MQTT discovery).
2. An **MQTT broker** add-on (e.g. *Mosquitto broker*) installed and started.
3. The **MQTT integration** enabled in Home Assistant.

By default the add-on discovers the broker host, port and credentials
automatically through the Supervisor. You can also point it to any external
broker with the `mqtt_*` options.

---

## Installation

1. Add this repository to the Home Assistant add-on store and install
   **Setecna REG PLUS**.
2. Open the add-on **Configuration** tab and fill in at least `systemID`,
   `username` and `password` (your s5a.eu account; the `systemID` is shown in the
   s5a.eu web interface once logged in).
3. Start the add-on. Your Setecna device and its zones appear automatically under
   **Settings -> Devices & Services -> MQTT**.

---

## Configuration

| Option | Default | Description |
|---|---|---|
| `systemID` | - | **Required.** Your system ID, from the s5a.eu web interface. |
| `username` | - | **Required.** Your s5a.eu account email. |
| `password` | - | **Required.** Your s5a.eu account password. |
| `readonly` | `false` | Only expose sensors; never write anything back to the plant. When `true`, no controls (switches, selects, numbers, thermostats) are created. |
| `adv_int` | `false` | Create a native `climate` (thermostat) entity for each active zone. Requires `readonly: false`. |
| `diagnostics` | `false` | Also expose the diagnostic device families (heat pumps, cascade controller, OpenTherm, relay outputs, system alarms). |
| `hide_unavailable` | `true` | Do not create sensors whose input reads a "not available" value on your system (they would otherwise show as *unknown*). Only genuinely unavailable channels are hidden. |
| `language` | `en` | UI language for entity labels and option names: `en`, `it`, `de`, `fr`, `es`. |
| `system_control` | `true` | Expose the master **System on/off** switch. |
| `season_control` | `true` | Expose the **Season** (winter/summer) selector. |
| `acs_control` | `true` | Expose the domestic-hot-water (ACS) controls. |
| `active_zones` | `[]` | If non-empty, only these zone numbers are exposed (e.g. `[1, 3, 6]`); empty means all detected zones. |
| `entity_names` | `[]` | Rename entities, one `PREFIX=Name` per entry (see *Renaming entities*). |
| `poll_interval` | `30` | Seconds between refreshes from the s5a.eu cloud (10-600). |
| `cleanup_legacy` | `true` | On startup, remove the per-entity discovery topics used by add-on v1.x. |
| `debug` | `false` | Verbose logging plus a full parameter dump; use only for support. |
| `mqtt_host` | - | Custom MQTT broker host/IP. Leave empty to auto-discover the broker via the Supervisor. |
| `mqtt_port` | `1883` | Custom broker port (only with `mqtt_host`). |
| `mqtt_username` | - | Custom broker username (empty = anonymous). |
| `mqtt_password` | - | Custom broker password. |

---

## What you get

### Main device

System-wide entities, including:

- **System on/off** switch and **Season** selector (winter/summer), when enabled.
- **Domestic hot water (ACS)**: enable, temperature, comfort/economy setpoints,
  hysteresis and delta, main output.
- **Dew point**, de-icing state and the zone hysteresis/threshold settings.
- **Analog inputs**, **digital inputs** and **alarms** (`ANY_ALARM` plus the
  individual alarm words).
- **Energy meters** (power and total imported/exported energy), when installed.
- **Calendars** (programmed clocks) and their forcing.
- **Last update** timestamp and the add-on **update** entity.
- **Diagnostic entities** (only with `diagnostics: true`): heat-pump units and
  cascade controller, OpenTherm generator cascade, board relay outputs.

### Per-zone sub-device

Each active zone becomes its own device with:

- **Temperature** and, if a probe is present, **humidity** and **dew point**.
- **State** (whether the zone is currently calling for heating/cooling).
- **Seasonal setpoints**: winter/summer comfort and economy.
- **Set humidity** number (target humidity), when a humidity probe is present.
- **Regime** sensor and **Forcing** select (see below).
- **Calendar** sensor (which programmed clock drives the zone).
- A native **thermostat** (`climate`) when Advanced integration is enabled.

---

## The thermostat

With `adv_int: true` (and `readonly: false`) each active zone gets a `climate`
entity:

- **Mode**: the season's `heat` (winter) or `cool` (summer) when the zone is on,
  and `off` when it is off. "Off" is shown whenever the zone is *actually* off -
  either forced off or off by its schedule - not only when you forced it.
  Changing the mode writes the forcing back to the plant (on -> automatic,
  off -> forced off).
- **Current temperature** and a single **target temperature** (the active
  seasonal setpoint).
- **Humidity**: when the zone has a humidity probe, the current and target
  humidity are shown on the thermostat over a full 0-100% scale.
- **Action**: heating / cooling / idle, reflecting the actual zone relay output.

### Why a single setpoint (Amazon Alexa)

The thermostat deliberately uses a **single setpoint** with `heat`/`cool` modes.
Amazon Alexa's `AUTO` and `ECO` thermostat modes require a *min/max* setpoint
range; exposing them (or a climate *preset*) makes Alexa spin without ever
showing the temperature and leaves an ECO badge stuck on. With a single setpoint
and no presets, temperature and on/off work correctly in Alexa. Alexa has no
"comfort" mode, so the comfort/economy regime is not exposed as an Alexa mode -
use the **Forcing** select instead (see next section).

> After changing anything that affects the thermostat, re-run Alexa device
> discovery ("Alexa, discover devices") so Alexa drops any cached capability.

---

## Regime vs Forcing (important)

These two per-zone entities look similar but mean different things:

- **Forcing** (select + sensor) - what *you* have manually forced. Its value is
  `automatic` when you have not forced anything, and `economy`/`comfort`/`off`
  only when you force it. This is the control you use to override the schedule.
- **Regime** (sensor) - the *actual* mode currently in effect, computed from the
  active setpoint: `automatic eco`, `automatic comfort`, `forced eco`,
  `forced comfort`, or `off`. This tells you whether the zone is in economy or
  comfort **even while running in automatic**, and works even when the zone is
  idle.

So to see if a zone is currently in economy or comfort, look at **Regime**; to
override it, use **Forcing**.

---

## Unavailable sensors (`hide_unavailable`)

A Setecna board reports a fixed "not available" sentinel for inputs that are not
wired or not installed (for example an external-temperature input with no probe,
or energy meters on a system without meters). Those channels have no real value,
so Home Assistant would show them as *unknown*.

With `hide_unavailable: true` (the default) such sensors are simply **not
created**. The filter only removes a sensor when its current value is a sentinel
that the sensor's own formula blanks out, so a genuine reading is never dropped,
and controls and computed sensors (regime, calendar) are never affected.

The check runs at startup: if an input that reads a sentinel today becomes valid
later, its sensor reappears after the next add-on restart. Set the option to
`false` if you prefer to see those channels as *unknown* (e.g. to diagnose
wiring).

---

## Master controls

`system_control`, `season_control` and `acs_control` let you hide the
system-wide controls you do not want exposed (for instance to prevent turning the
whole plant off by mistake). They only affect the main device; per-zone controls
are unaffected.

---

## Limiting zones (`active_zones`)

Large panels can define many zones. Set `active_zones` to the list of zone
numbers you actually use (e.g. `[1, 3, 6]`) to expose only those; leave it empty
to expose every detected zone.

---

## Renaming entities

Entities are named generically ("Zone 1 temperature", "Circuit 1 temperature",
...). Rename them from `entity_names`, one `PREFIX=Name` per entry:

```
Z1=Bagni
Z3=Soggiorno
C1=Panel mixing circuit
GLOBAL_OUTPUT_3=Recirculation pump
```

- A **prefix** (`Z1`, `C1`, `S1`, `HP0`, ...) renames every entity of that
  element and its thermostat in one go: `Z1=Bagni` gives "Bagni temperature",
  "Bagni dew point", the "Bagni" thermostat, and so on.
- An **exact parameter id** (e.g. `GLOBAL_OUTPUT_3`) renames a single entity.

To find out which zone is which, the add-on prints, on every start, the custom
labels you set on the Setecna panel and the description code of each active zone.
Look at the add-on log after startup and copy the labels into the option.
(Automatic naming from the panel's description codes is intentionally not done:
that dictionary is undocumented and guessing could mislabel rooms.)

`unique_id`s never change, so renaming here - or directly in the Home Assistant
UI - is preserved across restarts and updates.

---

## Energy meter 32-bit totals

Each energy accumulator is split by the controller into a low word (`ACCLO` /
`ACC2LO`, exposed as kWh) and a high word (`ACCHI` / `ACC2HI`). For totals that
exceed the 16-bit low word, reconstruct the full value with a template sensor:

```jinja
{{ (states('sensor.energy_meter_1_total_energy_import_high_word') | int * 65536
    + states('sensor.energy_meter_1_total_energy_import') | float * 10) / 10 }}
```

---

## MQTT topics

- Discovery (retained): `homeassistant/device/setecna_<systemID>/config`
- Availability (retained, Last Will): `setecna/<systemID>/availability`
- States (retained): `setecna/<systemID>/<PARAM>`
- Commands (from Home Assistant): `setecna/<systemID>/<PARAM>/set`

---

## Resilience

- Automatic re-authentication when the s5a.eu session expires.
- MQTT auto-reconnect; while the add-on is down all entities are marked
  `unavailable` via the Last Will message.
- Discovery and states are re-published on broker reconnect and whenever Home
  Assistant restarts (birth message).
- Cloud fetch failures back off exponentially (capped at 5 minutes).
- When the plant switches between winter and summer, thermostats are rebuilt
  with the correct seasonal setpoints.
- A self-update entity reports the running version and highlights newer GitHub
  releases.

---

## Upgrading from add-on v1.x

Entity `unique_id`s are unchanged, so entities, history and dashboards are
preserved. Raw state topics moved from `homeassistant/<type>/<systemID>_<PARAM>`
to `setecna/<systemID>/<PARAM>`; update any automation that read the raw MQTT
topics directly. On first start, `cleanup_legacy: true` removes the old
per-entity discovery topics; if some stale entities remain, restart Home
Assistant once.

---

## Troubleshooting

- **A control/thermostat shows a stale behaviour after an update** - Home
  Assistant sometimes keeps a cached MQTT discovery config. Delete the affected
  entity (or the whole Setecna device) and restart the add-on to recreate it.
- **Alexa shows an old mode/badge** - re-run Alexa device discovery; if it
  persists, remove the thermostat in the Alexa app and discover again.
- **A sensor you expect is missing** - it is probably an unavailable channel
  hidden by `hide_unavailable`. Check its raw MQTT topic value; if it reads a
  sentinel it is genuinely not present on your system.
