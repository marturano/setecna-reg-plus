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

## Funktionen

- **Ein Home-Assistant-Gerät pro Zone** (plus das Hauptgerät *Setecna REG* mit Globals, ACS, Kreisen, Quellen, Wärmepumpen und Regler): eine ganze Zone lässt sich über ihre Geräteseite umbenennen (erfordert Home Assistant **2024.11+**, getestet bis **2026.7**).
- **Hauptbedienelemente** (wenn beschreibbar): Anlage **ein/aus**, **Saison** (Winter/Sommer) und **ACS ein/aus**.
- **Native Klimaentitäten** (optionaler Modus *Erweiterte Integration*) für jede aktive Zone, mit Heiz-/Kühl-`hvac_action`, einer einzelnen Zieltemperatur (dem Comfort-Sollwert der Saison), übersetzten Presets (`eco`/`comfort`) und, sofern verfügbar, Feuchteregelung.
- **Zusätzliche Gerätefamilien** als schreibgeschützte Diagnose: Wärmepumpeneinheiten und Kaskadenregler, OpenTherm-Generatorkaskade (wenn aktiviert), Relaisausgänge der Platine, Systemalarme, Zonen-Taupunkt, Kreis-Rücklauftemperaturen und -Pumpen, Quellentemperaturen und 32-Bit-Energiezähler. Nicht verfügbare Kanäle bleiben *unbekannt*, statt Unsinnswerte anzuzeigen.
- **Umbenennen von Entitäten in den Add-on-Einstellungen**: eine Zone einmal benennen, alle ihre Entitäten übernehmen den Namen.
- **Verfügbarkeitsüberwachung** per MQTT Last Will: Entitäten werden `unavailable`, wenn das Add-on stoppt.
- **Selbstheilung**: automatisches erneutes Anmelden bei Ablauf der Setecna-Sitzung, automatische MQTT-Wiederverbindung mit Backoff, erneute Discovery-Veröffentlichung beim Neustart von Home Assistant, automatischer Neuaufbau der Klimaentitäten beim Wechsel Winter/Sommer.
- Unterstützung für einen **eigenen MQTT-Broker** oder automatische Erkennung des Mosquitto-Add-ons.
- **Update-Entität**, die die laufende Version anzeigt und auf neue GitHub-Releases hinweist.

## Installation

1. Stellen Sie sicher, dass Sie Home Assistant **2024.11 oder neuer** verwenden und ein MQTT-Broker (z. B. das Mosquitto-Broker-Add-on) samt MQTT-Integration eingerichtet ist.
2. Klicken Sie auf die Schaltfläche oben oder fügen Sie `https://github.com/marturano/setecna-reg-plus` unter *Einstellungen → Add-ons → Add-on-Store → ⋮ → Repositorys* hinzu.
3. Installieren Sie **Setecna REG PLUS**, füllen Sie die Konfiguration aus und starten Sie es.

## Konfiguration

| Option | Erforderlich | Beschreibung |
|---|---|---|
| `systemID` | ja | Ihre System-ID, nach der Anmeldung in der s5a.eu-Weboberfläche sichtbar |
| `username` | ja | E-Mail Ihres s5a.eu-Kontos |
| `password` | ja | Passwort Ihres s5a.eu-Kontos |
| `readonly` | nein (`false`) | Nur Sensoren bereitstellen; niemals ins System schreiben |
| `adv_int` | nein (`false`) | Native `climate`-Entitäten pro Zone erstellen (erfordert `readonly: false`) |
| `cleanup_legacy` | nein (`true`) | Per-Entität-Discovery-Topics der v1.x entfernen |
| `poll_interval` | nein (`30`) | Sekunden zwischen Aktualisierungen aus der Setecna-Cloud (10–600) |
| `mqtt_host` | nein | Eigener MQTT-Broker-Host. Leer = Mosquitto-Add-on automatisch erkennen |
| `mqtt_port` | nein (`1883`) | Port des eigenen Brokers (nur mit `mqtt_host`) |
| `mqtt_username` | nein | Benutzername des eigenen Brokers (leer = anonym) |
| `mqtt_password` | nein | Passwort des eigenen Brokers |
| `entity_names` | nein | Namensüberschreibungen, ein `PRÄFIX=Name` pro Eintrag (siehe unten) |
| `active_zones` | nein | Anzuzeigende Zonennummern; leer = alle erkannten Zonen |

### Entitäten umbenennen

Fügen Sie unter `entity_names` je einen `PRÄFIX=Name`-Eintrag hinzu:

```
Z1=Bäder
Z3=Wohnzimmer
C1=Mischkreis Paneele
GLOBAL_OUTPUT_3=Zirkulationspumpe
```

Ein **Zonen-Präfix** (`Z1`, `Z2`, ...) benennt das Gerät dieser Zone und damit alle Entitäten der Zone samt Thermostat auf einmal um; eine **exakte Parameter-ID** (z. B. `GLOBAL_OUTPUT_3`) benennt eine einzelne Entität auf dem Hauptgerät um. Präfixe und IDs berücksichtigen **Gross-/Kleinschreibung** (`Z1`, nicht `z1`). Beim Start protokolliert das Add-on die in Ihrem Setecna-Panel gespeicherten Bezeichnungen zum Kopieren.

Siehe [`setecna/DOCS.md`](setecna/DOCS.md) (auf Englisch) für MQTT-Topics, Diagnoseentitäten, Energiezähler-Summen, Update-Hinweise und Details zur Ausfallsicherheit.

## Migration vom ursprünglichen Add-on

Wenn Sie zuvor das ursprüngliche Add-on `homeassistant-addon-setecna` (1.x) von ingordigia verwendet haben, bleiben die `unique_id`s der Entitäten unverändert, sodass Entitäten, Verlauf und Dashboards erhalten bleiben. Die rohen State-Topics wandern nach `setecna/<systemID>/<PARAM>`; mit `cleanup_legacy: true` werden die alten Per-Entität-Discovery-Topics beim ersten Start entfernt. Wenn Sie das ursprüngliche Add-on nie verwendet haben, einfach installieren und loslegen. Vollständige Details im [Changelog](setecna/CHANGELOG.md).

---

## Danksagung

Dieses Add-on ist ein Fork des ursprünglichen Projekts [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) von **ingordigia**, in Go neu geschrieben und deutlich erweitert. Veröffentlicht unter der Apache-2.0-Lizenz.

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
