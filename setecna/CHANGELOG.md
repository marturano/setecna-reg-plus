<!-- https://developers.home-assistant.io/docs/add-ons/presentation#keeping-a-changelog -->

## 1.0.1

Maintenance release with thermostat, controls and layout fixes on top of 1.0.0.

### Fixed
- **Thermostat presets are now translated**: zone climate presets use Home Assistant's standard constants (`eco`, `comfort`, `none`) instead of custom English strings, so the frontend shows them in the user's language. The redundant, untranslatable "Zone N preset" select was removed - the thermostat's mode + preset already cover it.
- **Thermostat uses a single target temperature** (the comfort setpoint of the active season) instead of a low/high range. A range in a single (cool/heat) mode made Amazon Alexa (via Nabu Casa) hang on load and hide the temperature; a single setpoint fixes it. The economy setpoint stays adjustable via its own number entity.
- **System / Season / ACS on-off controls now actually write**: the Setecna cloud accepts writes to global parameters only on their `P_GLOBAL_*` name (while reading them as `GLOBAL_*`). These controls now write to `P_GLOBAL_ENABLE` / `P_GLOBAL_SEASON` / `P_GLOBAL_ACS_ENABLE`; state, unique_id and history are unchanged.

### Changed
- **ACS on its own device**: domestic-hot-water entities were renamed from "DHW" to "ACS" and moved to a dedicated `ACS` device, instead of being mixed with the global/system entities. Together with the per-zone/circuit/source/heat-pump devices this gives a clean split: *Setecna REG* (globals + system), *ACS*, and one device per element.
- **Heat-pump controller "required power"** is now reported in **kW** (device_class power), scaled from the raw hundredths-of-a-kW register (confirmed against 5 kW nominal stages).

## 1.0.0

First release of **Setecna REG PLUS**, an independent, ground-up rewrite (in Go) of the original `homeassistant-addon-setecna` by ingordigia. If you are migrating from that add-on, see *Migrating from the original add-on* in the documentation — entity `unique_id`s are preserved, so history and dashboards carry over.

### Added
- **Diagnostic entities are now disabled by default**: the raw/status/code sensors and alarm binary-sensors are still created but hidden, so the device pages stay clean. Enable any you want from the entity's settings. Primary measurements (temperatures, humidity, power, energy) stay enabled.
- **Master on/off control**: when the add-on is writable (readonly off), the plant on/off (`GLOBAL_ENABLE`) is exposed as a `switch` main control.
- **Season control**: the summer/winter selector (`GLOBAL_SEASON`) is exposed as a writable `select` (winter/summer) main control.
- **One Home Assistant device per element**: each active zone, circuit, source and heat pump is now its own device (linked to the main "Setecna REG" device), instead of a single device holding every entity. Entities are named by their measurement ("Temperature", "Dew point", ...) so Home Assistant composes "<device> <label>". This makes renaming native: rename the zone device (e.g. "Soggiorno") from its settings and every entity and the thermostat follow. Entity `unique_id`s are unchanged, so history and dashboards are preserved.
- **Zone allowlist** (`active_zones` option): expose only the zone numbers you actually use; zones detected on the panel but not listed - and all their sensors, controls and thermostat - are hidden and cleanly removed from Home Assistant.
- **Localisation**: the add-on configuration UI is now available in English, Italian, German, French and Spanish. The repository README is provided in the same five languages, with a language selector.
- **Entity renaming from the add-on settings** (`entity_names` option): rename entities with `PREFIX=Name` entries. A zone/circuit/source prefix (`Z1`, `C1`, `S1`, `HP0`) renames every entity of that element at once — e.g. `Z1=Bagni` turns "Zone 1 temperature" into "Bagni temperature" and the zone-1 thermostat into "Bagni" — while an exact parameter id (`GLOBAL_OUTPUT_3=Recirculation pump`) renames a single entity. On startup the add-on logs the custom labels stored in the Setecna system (`_FREEDESC`/`_XFREEDESC`) and each active zone's description code, to help fill in the mapping.
- **New device families exposed as read-only diagnostic entities** (discovered from a full parameter dump):
  - Heat-pump units `HP0..HP4` (present ones auto-detected via return temperature): return/flow/outside/DHW temperatures plus raw status, power, request and error codes.
  - Heat-pump cascade controller `HPC`: PID temperature (°C), active stages (count) and required power (kW, scaled from hundredths of a kW). Remaining fields (PID output, grace timer, requests, status/error codes, flags) are exposed as raw diagnostics since their scale/encoding is not documented.
  - OpenTherm generator cascade `OT_G0..OT_G8` (flow/DHW/return temperatures, status, power, error codes), created only when the OpenTherm subsystem and the individual generator are enabled.
  - Board relay outputs `GLOBAL_OUTPUT_0..15` as binary sensors.
  - System alarms: `ANY_ALARM` as a problem binary sensor plus the individual alarm words.
  - Zone dew point (`Z*_DEWPOINT`), circuit return temperature and pumps A/B (`C*_RET_TEMP`, `C*_OUTPUT_PA/PB`), source temperature/aux temperature/setpoint/status (`S*_TEMP`, `S*_AUXTEMP`, `S*_SET`, `S*_STATUS`).
  - Energy-meter accumulator high words (`EM*_ACCHI`, `EM*_ACC2HI`) so 32-bit totals can be reconstructed.
- Sentinel filtering (255 / 32768 / 32769 / 65280 / 65535) on all new fields: unavailable channels stay "unknown" instead of showing a garbage number.
- **MQTT device-based discovery** (Home Assistant 2024.11+): the whole system is now announced with a single retained message on `homeassistant/device/setecna_<systemID>/config` instead of hundreds of per-entity config topics.
- **Availability tracking**: `setecna/<systemID>/availability` with MQTT Last Will, so all entities become `unavailable` if the add-on stops.
- **Automatic re-login**: when the s5a.eu session expires the add-on now re-authenticates transparently instead of silently stopping updates.
- **Automatic recovery**: MQTT auto-reconnect, exponential backoff on cloud errors, discovery re-published when Home Assistant restarts (birth message on `homeassistant/status`).
- **Season switching at runtime**: when the system toggles between winter and summer, climate entities are rebuilt automatically without restarting the add-on.
- Retained state and discovery messages: entities and values survive broker and Home Assistant restarts.
- New options: `cleanup_legacy` (removes v1.x per-entity discovery topics on startup) and `poll_interval` (10-600 s).
- **Self-update entity**: reports the running add-on version and surfaces new GitHub releases (checked at startup and once a day).
- **Climate mode is now settable and coherent**: the HVAC mode follows the zone forcing state (off when "forced off", heat/cool otherwise) and can be changed from the thermostat card, which writes the forcing back to the system.
- **Optimistic command echo**: values changed from Home Assistant are reflected on the state topic immediately, without waiting for the next poll cycle.
- Explicit `startup: application` and `boot: auto` in the add-on manifest.
- **CI**: GitHub Actions workflow running gofmt, `go vet`, build and `go test -race`; dependabot now also tracks Go modules.
- **Custom MQTT broker support**: optional `mqtt_host`, `mqtt_port`, `mqtt_username`, `mqtt_password` options to use an external broker instead of the auto-discovered Mosquitto add-on (Supervisor service requirement relaxed from `need` to `want`).
- Verified compatible with Home Assistant 2026.7 (no use of `object_id`, removed in HA 2026.4, or other deprecated MQTT discovery options).
- Unit tests for the scraper and the discovery payload builder.

### Fixed
- **Enum sensors now include the required `options` list** (Global season, Zone/Calendar mode and preset). Without it, HA 2023.4+ rejected these entities and logged an error on every discovery. Enum sensors also no longer emit the incompatible `state_class`/`unit_of_measurement`.
- Climate entities now expose `hvac_action` (heating/cooling/idle) derived from the zone relay, so the thermostat card shows whether the zone is actively calling.

### Changed
- State/command topics moved from `homeassistant/<type>/<systemID>_<param>` to `setecna/<systemID>/<param>` (the `homeassistant/` prefix is reserved for discovery). **BREAKING** for anyone consuming the old raw topics; `unique_id`s are unchanged, so entities, history and dashboards migrate seamlessly.
- MQTT broker port is now read from the Supervisor service instead of being hard-coded to 1883.
- Structured logging (log/slog), graceful shutdown, single multi-arch Dockerfile, Go 1.24 build, paho.mqtt.golang v1.5.0.

### Removed
- Dependency on golang.org/x/net (CSRF token extraction is now self-contained).
- Dead code: unused config, event and water-heater models, per-arch Dockerfiles, tempio.

---

# Previous history — original `homeassistant-addon-setecna` by ingordigia

_Setecna REG PLUS is a fork of the project below; its version numbering restarts at 1.0.0. The entries here document the original add-on that this work is based on._

## 1.1.1

- Bump home-assistant/builder from 2024.08.1 to 2024.08.2
- Bump actions/checkout from 4.1.7 to 4.2.1
- Bump frenck/action-addon-linter from 2.15 to 2.17
- Bump actions/checkout from 4.2.1 to 4.2.2
- Bump frenck/action-addon-linter from 2.17 to 2.18
- Add network capability to make this run on HA Supervised on Debian 12

## 1.1.0

- Add MTx_MODE as a sensor and MTx_FORCING as a selector to HomeAssistant
- **BREAKING CHANGE**: Change how a zone is considered active, now the plugin check if Zx_SENSOR_CHN != 0 instead of Zx_TEMP != 32769 (aligned with the web interface logic)

## 1.0.1

- Fixes decimals in command template for climate entities

## 1.0.0

- Initial release
