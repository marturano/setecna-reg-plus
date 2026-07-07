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

## Funzionalità

- **Un device Home Assistant per zona** (più il device principale *Setecna REG* che raccoglie globali, ACS, circuiti, sorgenti, pompe di calore e controllore): puoi rinominare un'intera zona dalla sua pagina device (richiede Home Assistant **2024.11+**, verificato fino alla **2026.7**).
- **Controlli master** (quando scrivibile): **on/off** impianto, **stagione** (inverno/estate) e **on/off ACS**.
- **Entità climate native** (modalità *Integrazione avanzata*) per ogni zona attiva, con `hvac_action` riscaldamento/raffrescamento, una singola temperatura target (il setpoint comfort della stagione), preset tradotti (`eco`/`comfort`) e, dove disponibile, controllo umidità.
- **Famiglie di dispositivi aggiuntive** esposte come diagnostica in sola lettura: unità pompa di calore e controllore cascata, cascata generatori OpenTherm (quando abilitata), uscite relè della scheda, allarmi di sistema, punto di rugiada di zona, temperature di ritorno e pompe dei circuiti, temperature delle sorgenti e totalizzatori energia a 32 bit. I canali non disponibili restano *sconosciuti* invece di mostrare valori spuri.
- **Rinomina delle entità dalle impostazioni dell'add-on**: dai il nome a una zona e tutte le sue entità lo ereditano.
- **Tracciamento disponibilità** tramite Last Will MQTT: le entità diventano `unavailable` se l'add-on si ferma.
- **Auto-recupero**: re-login automatico alla scadenza della sessione Setecna, riconnessione MQTT con backoff, discovery ripubblicata al riavvio di Home Assistant, entità climate ricostruite al cambio stagione.
- **Broker MQTT personalizzato** oppure rilevamento automatico dell'add-on Mosquitto.
- **Entità di aggiornamento** che mostra la versione in uso e segnala nuove release su GitHub.

## Installazione

1. Verifica di usare Home Assistant **2024.11 o successivo** con un broker MQTT (es. add-on Mosquitto) e l'integrazione MQTT configurata.
2. Usa il pulsante qui sopra, oppure aggiungi `https://github.com/marturano/setecna-reg-plus` in *Impostazioni → Add-on → Store → ⋮ → Repository*.
3. Installa **Setecna REG PLUS**, compila la configurazione e avvialo.

## Configurazione

| Opzione | Obbligatoria | Descrizione |
|---|---|---|
| `systemID` | sì | ID del sistema, visibile nell'interfaccia web s5a.eu dopo il login |
| `username` | sì | Email dell'account s5a.eu |
| `password` | sì | Password dell'account s5a.eu |
| `readonly` | no (`false`) | Espone solo i sensori; non scrive mai sul sistema |
| `adv_int` | no (`false`) | Crea entità `climate` native per zona (richiede `readonly: false`) |
| `cleanup_legacy` | no (`true`) | Rimuove i topic di discovery per-entità delle versioni 1.x |
| `poll_interval` | no (`30`) | Secondi tra un aggiornamento e l'altro dal cloud Setecna (10–600) |
| `mqtt_host` | no | Host broker MQTT personalizzato. Vuoto = usa l'add-on Mosquitto |
| `mqtt_port` | no (`1883`) | Porta del broker personalizzato (solo con `mqtt_host`) |
| `mqtt_username` | no | Nome utente del broker personalizzato (vuoto = anonimo) |
| `mqtt_password` | no | Password del broker personalizzato |
| `entity_names` | no | Rinomina entità, una voce `PREFISSO=Nome` (vedi sotto) |
| `active_zones` | no | Numeri di zona da esporre; vuoto = tutte le zone rilevate |

### Rinomina delle entità

Aggiungi una voce `PREFISSO=Nome` per riga in `entity_names`:

```
Z1=Bagni
Z3=Soggiorno
C1=Miscelato pannelli
GLOBAL_OUTPUT_3=Pompa ricircolo
```

Un **prefisso di zona** (`Z1`, `Z2`, ...) rinomina il device di quella zona e, con esso, tutte le entità della zona e il suo termostato in un colpo solo; un **id di parametro esatto** (es. `GLOBAL_OUTPUT_3`) rinomina una singola entità sul device principale. Prefissi e id fanno **distinzione tra maiuscole e minuscole** (usa `Z1`, non `z1`). All'avvio l'add-on scrive nel log le etichette personalizzate del tuo pannello Setecna da copiare.

Vedi [`setecna/DOCS.md`](setecna/DOCS.md) (in inglese) per topic MQTT, entità diagnostiche, totalizzatori energia, note di aggiornamento e dettagli sulla resilienza.

## Migrazione dall'add-on originale

Se usavi l'add-on originale `homeassistant-addon-setecna` (1.x) di ingordigia, i `unique_id` delle entità restano invariati: entità, storico e dashboard si conservano. I topic di stato grezzi passano a `setecna/<systemID>/<PARAM>`; con `cleanup_legacy: true` i vecchi topic per-entità vengono rimossi al primo avvio. Se non hai mai usato l'add-on originale, installa e via. Dettagli completi nel [changelog](setecna/CHANGELOG.md).

---

## Crediti

Questo add-on è un fork del progetto originale [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) di **ingordigia**, riscritto in Go e ampiamente esteso. Distribuito con licenza Apache-2.0.

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
