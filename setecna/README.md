# Home Assistant Add-on: Setecna REG PLUS

_Integrates a Setecna REG thermal system into Home Assistant over MQTT (device-based discovery)._

![Supports aarch64 Architecture][aarch64-shield]
![Supports amd64 Architecture][amd64-shield]

Bridges a Setecna REG plant (through the **s5a.eu** cloud) to Home Assistant. The whole system appears as a main device plus one sub-device per zone.

**Highlights**

- Per-zone temperature, humidity, dew point, state and seasonal setpoints.
- Optional native **thermostat** per zone (single setpoint, Alexa-friendly), with humidity on a 0-100% scale.
- **Regime** sensor (live comfort/eco, automatic or forced) and **Forcing** select.
- Domestic hot water, circuits, sources, energy meters, calendars and optional diagnostics (heat pumps, OpenTherm, relays, alarms).
- **`hide_unavailable`** hides not-installed channels; entity renaming; five UI languages.

Set at least `systemID`, `username` and `password`, then start the add-on.

See **[`DOCS.md`](DOCS.md)** for the full documentation, and the repository README for a translated overview: [English](../README.md) · [Italiano](../README.it.md) · [Deutsch](../README.de.md) · [Français](../README.fr.md) · [Español](../README.es.md).

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
