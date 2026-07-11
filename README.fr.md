# AVERTISSEMENT

Il s'agit d'une intégration non officielle, développée par la communauté. Elle **n'est en aucune manière affiliée, approuvée, sponsorisée ni prise en charge par SETECNA EPC Srl**, fournisseur du système Setecna REG et du portail de télégestion s5a.eu (www.s5a.eu).

« Setecna », « Setecna REG », « s5a.eu » ainsi que les noms ou logos associés sont des marques ou la propriété de leurs détenteurs respectifs, utilisés ici uniquement à des fins d'identification et d'interopérabilité. Ce projet est développé de manière indépendante, par rétro-ingénierie de l'interface web s5a.eu accessible publiquement.

**Aucune garantie.** Le logiciel est fourni « en l'état », sans garantie d'aucune sorte, expresse ou implicite, y compris la qualité marchande, l'adéquation à un usage particulier, l'exactitude ou l'absence de contrefaçon.

**Limitation de responsabilité.** Dans la mesure maximale permise par la loi, le ou les auteurs déclinent toute responsabilité pour tout dommage, perte ou coût de quelque nature que ce soit - direct, indirect, accessoire, spécial ou consécutif - découlant de l'utilisation ou de l'impossibilité d'utiliser cette intégration. Cela inclut notamment : dysfonctionnement, dommage ou usure de l'installation de chauffage/refroidissement ou de ses composants ; lectures ou commandes erronées, manquantes ou retardées ; perte de confort, gaspillage d'énergie ou surcoûts ; perte de données ; interruption ou suspension du service s5a.eu ou de votre compte.

**Utilisation à vos risques.** L'intégration peut lire et, si activé, **écrire** des paramètres sur votre système Setecna REG (consignes, saison, marche/arrêt, forçages, ...). Ces écritures modifient le comportement réel de votre installation. Vous êtes seul responsable de son utilisation et de toute modification apportée à votre système. Cette intégration **n'est pas un dispositif de sécurité** et ne doit être utilisée pour aucune fonction critique de sécurité.

**Conditions des tiers.** Il vous incombe de veiller à ce que votre utilisation respecte les conditions de service de SETECNA EPC Srl relatives au système Setecna REG et au portail s5a.eu. L'accès peut être modifié ou retiré par le fournisseur à tout moment, ce qui peut empêcher le fonctionnement de cette intégration.

# Dépôt de l'add-on Home Assistant pour Setecna REG

> 🌐 **Langues :** 🇬🇧 [English](README.md) · 🇮🇹 [Italiano](README.it.md) · 🇩🇪 [Deutsch](README.de.md) · 🇫🇷 Français · 🇪🇸 [Español](README.es.md)

Ce dépôt contient l'add-on Home Assistant qui intègre un système thermique **Setecna REG** dans Home Assistant via MQTT, en utilisant la [découverte basée sur les appareils](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery).

[![Ouvrez votre instance Home Assistant et affichez la boîte de dialogue d'ajout d'un dépôt d'add-on.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fmarturano%2Fsetecna-reg-plus)

![Prend en charge l'architecture aarch64][aarch64-shield]
![Prend en charge l'architecture amd64][amd64-shield]

---

## Fonctionnement

Les systemes Setecna REG se gerent via le portail cloud **s5a.eu**. L'add-on s'y connecte, recupere a intervalle regulier l'instantane complet du systeme et le republie dans Home Assistant via **MQTT** (decouverte basee sur l'appareil); les commandes de Home Assistant sont retraduites en ecritures sur le portail.

```
Installation Setecna REG  <->  cloud s5a.eu  <->  [ add-on ]  <->  broker MQTT  <->  Home Assistant
```

L'installation apparait comme un **appareil principal** (capteurs et commandes systeme) plus **un sous-appareil par zone active**. Les `unique_id` des entites sont stables : historique, tableaux de bord et renommages manuels survivent aux redemarrages et mises a jour.

## Fonctionnalites

- **Un appareil Home Assistant par zone**, plus l'appareil principal *Setecna REG* (globales, ECS, circuits, sources, compteurs d'energie, calendriers et diagnostics optionnels). Necessite Home Assistant **2024.11+**.
- **Thermostat natif** par zone (optionnel) : `heat`/`cool` a consigne unique + `off`, temperature actuelle et humidite actuelle/reglee (0-100%) si une sonde est presente. Optimise pour **Amazon Alexa**.
- Capteur **Regime** (automatique/force x confort/eco en temps reel, correct meme zone au repos) et selecteur **Forcing** pour outrepasser le programme.
- **Arret affiche quand la zone est reellement arretee** (forcee ou par programme), pas seulement en forcage manuel.
- **Calendriers par zone**, consignes saisonnieres confort/economie, point de rosee, etat de zone.
- **Entites de diagnostic** (optionnelles) : pompes a chaleur, regulateur cascade, cascade OpenTherm, sorties relais, alarmes.
- **`hide_unavailable`** (active par defaut) : les canaux non cables/installes sont masques au lieu d'apparaitre en *inconnu*.
- **Renommage d'entites**, **commandes maitres** (systeme/saison/ECS), **filtre de zones**, **cinq langues** (en/it/de/fr/es).
- **Resilient** : reconnexion au cloud, reconnexion MQTT avec disponibilite (Last Will), republication au redemarrage de HA, backoff sur erreurs, entite d'auto-mise a jour.

## Installation

1. Ajoutez ce depot au magasin d'add-ons et installez **Setecna REG PLUS**.
2. Dans la **Configuration**, renseignez au moins `systemID`, `username` et `password` (compte s5a.eu).
3. Demarrez l'add-on. L'appareil et ses zones apparaissent dans **Parametres -> Appareils et services -> MQTT**.

Un broker MQTT (ex. *Mosquitto broker*) et l'integration MQTT sont requis. Le broker est detecte automatiquement via le Superviseur, ou defini manuellement avec les options `mqtt_*`.

## Configuration

Les options les plus utiles (tableau complet et details dans [`setecna/DOCS.md`](setecna/DOCS.md)):

| Option | Defaut | Description |
|---|---|---|
| `systemID` / `username` / `password` | - | Identifiants s5a.eu **requis**. |
| `readonly` | `false` | Capteurs seuls ; aucune commande. |
| `adv_int` | `false` | Thermostat par zone (necessite `readonly: false`). |
| `diagnostics` | `false` | Pompes a chaleur / OpenTherm / relais / alarmes. |
| `hide_unavailable` | `true` | Masque les canaux indisponibles au lieu de *inconnu*. |
| `language` | `en` | `en` / `it` / `de` / `fr` / `es`. |
| `active_zones` | `[]` | Limite les zones exposees (vide = toutes). |
| `entity_names` | `[]` | Renomme les entites (`Z1=Bagni`, ...). |
| `poll_interval` | `30` | Secondes entre rafraichissements cloud (10-600). |

Documentation complete dans **[`setecna/DOCS.md`](setecna/DOCS.md)**.

## Migration depuis l'add-on original

Les `unique_id` sont inchanges : entites, historique et tableaux de bord sont preserves. Au premier demarrage, `cleanup_legacy` supprime les anciens topics de decouverte par entite ; s'il reste des entites obsoletes, redemarrez Home Assistant une fois.

## Crédits

Cet add-on est un fork du projet original [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) par **ingordigia**, réécrit en Go et largement étendu. Distribué sous licence Apache-2.0.

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
