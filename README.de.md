# HAFTUNGSAUSSCHLUSS

Dies ist eine inoffizielle, von der Community entwickelte Integration. Sie steht **in keiner Weise in Verbindung mit SETECNA EPC Srl und wird von dieser weder unterstützt, empfohlen noch gefördert** - dem Anbieter des Setecna-REG-Systems und des Fernwartungsportals s5a.eu (www.s5a.eu).

"Setecna", "Setecna REG", "s5a.eu" sowie zugehörige Namen oder Logos sind Marken bzw. Eigentum ihrer jeweiligen Inhaber und werden hier ausschliesslich zur Identifikation und Interoperabilität verwendet. Das Projekt wird unabhängig entwickelt, durch Reverse Engineering der öffentlich erreichbaren s5a.eu-Weboberfläche.

**Keine Gewährleistung.** Die Software wird "wie besehen" bereitgestellt, ohne jegliche ausdrückliche oder stillschweigende Gewährleistung, einschliesslich Marktgängigkeit, Eignung für einen bestimmten Zweck, Richtigkeit oder Nichtverletzung von Rechten.

**Haftungsbeschränkung.** Soweit gesetzlich zulässig, übernehmen die Autor(en) keine Haftung für Schäden, Verluste oder Kosten jeglicher Art - direkt, indirekt, beiläufig, speziell oder Folgeschäden - die aus der Nutzung oder Nichtnutzbarkeit dieser Integration entstehen. Dazu gehören ohne Einschränkung: Fehlfunktion, Beschädigung oder Verschleiss der Heiz-/Kühlanlage oder ihrer Komponenten; falsche, fehlende oder verzögerte Messwerte und Befehle; Komforteinbussen, Energieverschwendung oder höhere Kosten; Datenverlust; Unterbrechung oder Sperrung des s5a.eu-Dienstes oder Ihres Kontos.

**Nutzung auf eigene Gefahr.** Die Integration kann Parameter des Setecna-REG-Systems lesen und, sofern aktiviert, **schreiben** (Sollwerte, Saison, Ein/Aus, Zwang, ...). Solche Schreibvorgänge verändern das tatsächliche Verhalten Ihrer Anlage. Sie allein sind für die Nutzung und für jede vorgenommene Änderung verantwortlich. Diese Integration ist **kein Sicherheitsgerät** und darf für keine sicherheitskritische Funktion herangezogen werden.

**Bedingungen Dritter.** Sie sind dafür verantwortlich, dass Ihre Nutzung den Nutzungsbedingungen von SETECNA EPC Srl für das Setecna-REG-System und das s5a.eu-Portal entspricht. Der Zugang kann vom Anbieter jederzeit geändert oder entzogen werden, wodurch diese Integration nicht mehr funktionieren kann.

# Home-Assistant-Add-on-Repository für Setecna REG

> 🌐 **Sprachen:** 🇬🇧 [English](README.md) · 🇮🇹 [Italiano](README.it.md) · 🇩🇪 Deutsch · 🇫🇷 [Français](README.fr.md) · 🇪🇸 [Español](README.es.md)

Dieses Repository enthält das Home-Assistant-Add-on, das ein **Setecna-REG**-Wärmesystem über MQTT mittels [gerätebasierter Discovery](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery) in Home Assistant integriert.

[![Öffnen Sie Ihre Home-Assistant-Instanz und zeigen Sie den Dialog zum Hinzufügen eines Add-on-Repositorys an.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fmarturano%2Fsetecna-reg-plus)

![Unterstützt aarch64-Architektur][aarch64-shield]
![Unterstützt amd64-Architektur][amd64-shield]

---

## Funktionsweise

Setecna-REG-Systeme werden ueber das Cloud-Portal **s5a.eu** verwaltet. Das Add-on meldet sich dort an, ruft in festen Abstaenden den vollstaendigen System-Snapshot ab und veroeffentlicht alles per **MQTT** (geraetebasierte Discovery) in Home Assistant; Befehle aus Home Assistant werden in Portal-Schreibvorgaenge uebersetzt.

```
Setecna-REG-Anlage  <->  s5a.eu-Cloud  <->  [ Add-on ]  <->  MQTT-Broker  <->  Home Assistant
```

Die Anlage erscheint als **Hauptgeraet** (systemweite Sensoren und Bedienelemente) plus **ein Untergeraet je aktiver Zone**. Die `unique_id`s der Entitaeten sind stabil, daher bleiben Verlauf, Dashboards und manuelle Umbenennungen ueber Neustarts und Updates erhalten.

## Funktionen

- **Ein Home-Assistant-Geraet pro Zone**, plus das Hauptgeraet *Setecna REG* (Globals, WW, Kreise, Quellen, Energiezaehler, Kalender und optionale Diagnose). Benoetigt Home Assistant **2024.11+**.
- **Natives Thermostat** pro Zone (optional): `heat`/`cool` mit einem Sollwert + `off`, aktuelle Temperatur und aktuelle/eingestellte Feuchte (0-100%) bei vorhandener Sonde. Fuer **Amazon Alexa** optimiert.
- **Regime**-Sensor (live automatisch/erzwungen x Komfort/Eco, auch bei ruhender Zone korrekt) und **Forcing**-Auswahl zum Ueberschreiben des Programms.
- **Aus wird angezeigt, wenn die Zone wirklich aus ist** (erzwungen oder per Programm), nicht nur bei manueller Erzwingung.
- **Kalender je Zone**, saisonale Komfort-/Eco-Sollwerte, Taupunkt, Zonenstatus.
- **Diagnose-Entitaeten** (optional): Waermepumpen, Kaskadenregler, OpenTherm-Kaskade, Relaisausgaenge, Alarme.
- **`hide_unavailable`** (standardmaessig an): nicht verdrahtete/installierte Kanaele werden ausgeblendet statt als *unbekannt* zu erscheinen.
- **Entitaeten umbenennen**, **Master-Steuerungen** (System/Saison/WW), **Zonenfilter**, **fuenf Sprachen** (en/it/de/fr/es).
- **Robust**: Cloud-Neuanmeldung, MQTT-Reconnect mit Verfuegbarkeit (Last Will), erneute Veroeffentlichung beim HA-Neustart, Backoff bei Fehlern, Self-Update-Entitaet.

## Installation

1. Dieses Repository zum Add-on-Store hinzufuegen und **Setecna REG PLUS** installieren.
2. In der **Konfiguration** mindestens `systemID`, `username` und `password` (s5a.eu-Konto) setzen.
3. Add-on starten. Geraet und Zonen erscheinen unter **Einstellungen -> Geraete & Dienste -> MQTT**.

Ein MQTT-Broker (z. B. *Mosquitto broker*) und die MQTT-Integration sind erforderlich. Der Broker wird automatisch ueber den Supervisor erkannt oder manuell mit den `mqtt_*`-Optionen gesetzt.

## Konfiguration

Die wichtigsten Optionen (vollstaendige Tabelle und Details in [`setecna/DOCS.md`](setecna/DOCS.md)):

| Option | Standard | Beschreibung |
|---|---|---|
| `systemID` / `username` / `password` | - | **Erforderliche** s5a.eu-Zugangsdaten. |
| `readonly` | `false` | Nur Sensoren; keine Bedienelemente. |
| `adv_int` | `false` | Thermostat je Zone (benoetigt `readonly: false`). |
| `diagnostics` | `false` | Waermepumpen / OpenTherm / Relais / Alarme. |
| `hide_unavailable` | `true` | Nicht verfuegbare Kanaele ausblenden statt *unbekannt*. |
| `language` | `en` | `en` / `it` / `de` / `fr` / `es`. |
| `active_zones` | `[]` | Zonen einschraenken (leer = alle). |
| `entity_names` | `[]` | Entitaeten umbenennen (`Z1=Bagni`, ...). |
| `poll_interval` | `30` | Sekunden zwischen Cloud-Aktualisierungen (10-600). |

Vollstaendige Dokumentation in **[`setecna/DOCS.md`](setecna/DOCS.md)**.

## Migration vom ursprünglichen Add-on

Die `unique_id`s bleiben unveraendert, daher bleiben Entitaeten, Verlauf und Dashboards erhalten. Beim ersten Start entfernt `cleanup_legacy` die alten Per-Entitaet-Discovery-Topics; falls veraltete Entitaeten bleiben, Home Assistant einmal neu starten.

## Danksagung

Dieses Add-on ist ein Fork des ursprünglichen Projekts [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) von **ingordigia**, in Go neu geschrieben und deutlich erweitert. Veröffentlicht unter der Apache-2.0-Lizenz.

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
