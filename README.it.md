# AVVERTENZA

Questa è un'integrazione non ufficiale, sviluppata dalla community. **Non è affiliata, approvata, sponsorizzata né supportata in alcun modo da SETECNA EPC Srl**, fornitrice del sistema Setecna REG e del portale di telegestione s5a.eu (www.s5a.eu).

"Setecna", "Setecna REG", "s5a.eu" e ogni nome o logo correlato sono marchi o proprietà dei rispettivi titolari, qui utilizzati solo a fini di identificazione e interoperabilità. Il progetto è sviluppato in modo indipendente, tramite reverse engineering dell'interfaccia web di s5a.eu pubblicamente raggiungibile.

**Nessuna garanzia.** Il software è fornito "così com'è", senza garanzie di alcun tipo, esplicite o implicite, incluse a titolo esemplificativo commerciabilità, idoneità a uno scopo particolare, accuratezza o assenza di violazioni.

**Esclusione di responsabilità.** Nella misura massima consentita dalla legge, l'autore/gli autori non si assumono alcuna responsabilità per danni, perdite o costi di qualsiasi natura - diretti, indiretti, incidentali, speciali o consequenziali - derivanti dall'uso o dall'impossibilità di usare questa integrazione. Ciò include, senza limitazione: malfunzionamento, danno o usura dell'impianto di riscaldamento/raffrescamento o dei suoi componenti; letture o comandi errati, mancati o ritardati; perdita di comfort, spreco energetico o maggiori costi; perdita di dati; interruzione o sospensione del servizio s5a.eu o del proprio account.

**Uso a proprio rischio.** L'integrazione può leggere e, se abilitato, **scrivere** parametri sul sistema Setecna REG (setpoint, stagione, on/off, forzature, ...). Tali scritture modificano il comportamento reale dell'impianto. L'utente è l'unico responsabile dell'uso che ne fa e di ogni modifica apportata al proprio sistema. Questa integrazione **non è un dispositivo di sicurezza** e non deve essere utilizzata per alcuna funzione critica per la sicurezza.

**Termini di terze parti.** È responsabilità dell'utente assicurarsi che l'uso sia conforme ai termini di servizio di SETECNA EPC Srl relativi al sistema Setecna REG e al portale s5a.eu. L'accesso può essere modificato o revocato dal fornitore in qualsiasi momento, impedendo il funzionamento dell'integrazione.

# Repository add-on Home Assistant per Setecna REG

> 🌐 **Lingue:** 🇬🇧 [English](README.md) · 🇮🇹 Italiano · 🇩🇪 [Deutsch](README.de.md) · 🇫🇷 [Français](README.fr.md) · 🇪🇸 [Español](README.es.md)

Questo repository contiene l'add-on che integra un sistema termico **Setecna REG** in Home Assistant via MQTT, usando la [discovery basata su device](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery).

[![Apri la tua istanza Home Assistant e mostra la finestra per aggiungere un repository di add-on.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fmarturano%2Fsetecna-reg-plus)

![Supporta architettura aarch64][aarch64-shield]
![Supporta architettura amd64][amd64-shield]

---

## Come funziona

I sistemi Setecna REG si gestiscono tramite il portale cloud **s5a.eu**. L'add-on accede al portale, preleva a intervalli regolari lo snapshot completo del sistema e lo ripubblica in Home Assistant via **MQTT** con la discovery basata su dispositivo; i comandi da Home Assistant vengono ritradotti in scritture sul portale.

```
Impianto Setecna REG  <->  cloud s5a.eu  <->  [ add-on ]  <->  broker MQTT  <->  Home Assistant
```

L'impianto compare come **dispositivo principale** (sensori e controlli di sistema) piu **un sotto-dispositivo per ogni zona attiva**. Gli `unique_id` delle entita sono stabili, quindi storico, dashboard e rinomine manuali sopravvivono a riavvii e aggiornamenti.

## Funzionalita

- **Un dispositivo Home Assistant per zona**, piu il dispositivo principale *Setecna REG* (globali, ACS, circuiti, sorgenti, contatori energia, calendari e diagnostica opzionale). Richiede Home Assistant **2024.11+**.
- **Termostato nativo** per zona (opzionale): `heat`/`cool` a setpoint singolo + `off`, temperatura corrente e umidita corrente/impostata (0-100%) se presente la sonda. Ottimizzato per funzionare con **Amazon Alexa**.
- Sensore **Regime** (automatico/forzato x comfort/eco in tempo reale, corretto anche a zona ferma) e selettore **Forzatura** per scavalcare il programma.
- **Spento mostrato quando la zona e davvero spenta** (forzata o da programma), non solo quando forzata a mano.
- **Calendari per zona**, setpoint stagionali comfort/economia, punto di rugiada, stato zona.
- **Entita diagnostiche** (opzionali): pompe di calore, controllore a cascata, cascata OpenTherm, uscite rele, allarmi.
- **`hide_unavailable`** (attivo di default): i canali non cablati/installati sul tuo impianto vengono nascosti invece di apparire come *sconosciuto*.
- **Rinomina entita**, **controlli master** (sistema/stagione/ACS), **filtro zone**, **cinque lingue** (en/it/de/fr/es).
- **Resiliente**: ri-login al cloud, riconnessione MQTT con disponibilita (Last Will), ripubblicazione al riavvio di HA, backoff sugli errori, entita di auto-aggiornamento.

## Installazione

1. Aggiungi questo repository allo store degli add-on e installa **Setecna REG PLUS**.
2. Nella **Configurazione** dell'add-on imposta almeno `systemID`, `username` e `password` (il tuo account s5a.eu).
3. Avvia l'add-on. Il dispositivo Setecna e le sue zone compaiono in **Impostazioni -> Dispositivi e servizi -> MQTT**.

Servono un broker MQTT (es. *Mosquitto broker*) e l'integrazione MQTT abilitata. Il broker viene rilevato automaticamente dal Supervisor, oppure lo imposti a mano con le opzioni `mqtt_*`.

## Configurazione

Le opzioni piu utili (tabella completa e spiegazioni dettagliate in [`setecna/DOCS.md`](setecna/DOCS.md)):

| Opzione | Default | Descrizione |
|---|---|---|
| `systemID` / `username` / `password` | - | Credenziali s5a.eu **obbligatorie**. |
| `readonly` | `false` | Solo sensori; nessun controllo creato. |
| `adv_int` | `false` | Crea un termostato per zona (richiede `readonly: false`). |
| `diagnostics` | `false` | Espone pompe di calore / OpenTherm / rele / allarmi. |
| `hide_unavailable` | `true` | Nasconde i canali non disponibili invece di mostrare *sconosciuto*. |
| `language` | `en` | `en` / `it` / `de` / `fr` / `es`. |
| `active_zones` | `[]` | Limita quali zone esporre (vuoto = tutte). |
| `entity_names` | `[]` | Rinomina le entita (`Z1=Bagni`, ...). |
| `poll_interval` | `30` | Secondi tra i refresh dal cloud (10-600). |

Documentazione completa (comportamento del termostato, differenza Regime/Forzatura, note Alexa, rinomina e risoluzione problemi) in **[`setecna/DOCS.md`](setecna/DOCS.md)**.

## Migrazione dall'add-on originale

Gli `unique_id` delle entita non cambiano, quindi entita, storico e dashboard sono preservati. Al primo avvio `cleanup_legacy` rimuove i vecchi topic di discovery per-entita; se restano entita obsolete, riavvia Home Assistant una volta.

## Crediti

Questo add-on è un fork del progetto originale [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) di **ingordigia**, riscritto in Go e ampiamente esteso. Distribuito con licenza Apache-2.0.

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
