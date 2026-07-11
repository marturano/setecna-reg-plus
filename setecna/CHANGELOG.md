<!-- https://developers.home-assistant.io/docs/add-ons/presentation#keeping-a-changelog -->

## 1.1.0

First stable release of **Setecna REG PLUS**. The add-on bridges a Setecna REG
thermal system (via the s5a.eu cloud) to Home Assistant over MQTT, using
device-based discovery. This entry consolidates all the work from the earlier
pre-releases into a single stable baseline.

### Integration
- **MQTT device-based discovery** (Home Assistant 2024.11+): the whole plant
  appears as a main device, with each active zone as its own sub-device.
- **Automatic MQTT broker discovery** through the Supervisor, or a manual
  broker via `mqtt_host` / `mqtt_port` / `mqtt_username` / `mqtt_password`.
- **Resilience**: automatic s5a.eu re-login, MQTT auto-reconnect with a Last
  Will availability topic, discovery/state re-published on broker reconnect and
  on Home Assistant restart, exponential backoff on cloud fetch failures.
- **Self-update entity** reporting the running version and newer GitHub releases.
- **Five UI languages** for entity labels and options (en/it/de/fr/es).

### Zones and thermostat
- Per-zone temperature, humidity, relay state, dew point and seasonal setpoints
  (winter/summer, comfort/economy).
- Native **thermostat** (`climate`) per active zone (Advanced integration mode):
  single-setpoint `heat`/`cool` mode plus `off`, current temperature and, when a
  humidity probe is present, current + target humidity on a 0-100% scale. Tuned
  for Amazon Alexa (single setpoint, no presets) so temperature and on/off work
  reliably there.
- The thermostat shows **off whenever the zone is actually off** - forced off or
  off by schedule - not only when manually forced.
- **Regime** sensor showing the live mode (automatic/forced x comfort/eco, or
  off), derived from the active setpoint so it is correct even when the zone is
  idle. **Forcing** select to force automatic / off / economy / comfort.
- **Per-zone calendar** sensor (which clock/program drives the zone).

### Other devices
- Domestic hot water (ACS), circuits, sources, dehumidifiers, analog/digital
  inputs, alarms, energy meters and calendars.
- Optional **diagnostic entities** (`diagnostics: true`): heat-pump units and
  cascade controller, OpenTherm generator cascade, board relay outputs and
  system alarms.

### Configuration
- `readonly`, `adv_int` (thermostats), `diagnostics`, `language`,
  `system_control` / `season_control` / `acs_control` (master controls),
  `active_zones` (limit which zones are exposed), `entity_names` (rename
  entities), `poll_interval`, `cleanup_legacy`, `debug`.
- **`hide_unavailable`** (on by default): sensors whose input reads a
  "not available" sentinel on your system are not created, instead of appearing
  as *unknown*. Only genuinely unavailable channels are hidden; wired inputs are
  never affected.

### Notes for Amazon Alexa
- The thermostat is a single-setpoint `heat`/`cool` device. Alexa's AUTO/ECO
  modes require a min/max range and are intentionally not used, so temperature
  and on/off behave correctly. The comfort/economy regime is controlled by the
  separate **Forcing** select and shown by the **Regime** sensor, not as an
  Alexa thermostat mode (Alexa has no "comfort" mode).
