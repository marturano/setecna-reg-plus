# AVERTISSEMENT

Cet add-on est développé par rétro-ingénierie de l'interface web Setecna et n'est **pas** officiellement pris en charge par l'équipe Setecna. Utilisez-le à vos risques et périls.

# Dépôt de l'add-on Home Assistant pour Setecna REG

> 🌐 **Langues :** 🇬🇧 [English](README.md) · 🇮🇹 [Italiano](README.it.md) · 🇩🇪 [Deutsch](README.de.md) · 🇫🇷 Français · 🇪🇸 [Español](README.es.md)

Ce dépôt contient l'add-on Home Assistant qui intègre un système thermique **Setecna REG** dans Home Assistant via MQTT, en utilisant la [découverte basée sur les appareils](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery).

[![Ouvrez votre instance Home Assistant et affichez la boîte de dialogue d'ajout d'un dépôt d'add-on.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fmarturano%2Fsetecna-reg-plus)

![Prend en charge l'architecture aarch64][aarch64-shield]
![Prend en charge l'architecture amd64][amd64-shield]

---

## Fonctionnalités

- **Un appareil Home Assistant par élément** (appareil principal *Setecna REG* plus un par zone, circuit, source, pompe à chaleur et ACS) : les entités sont regroupées et une zone entière peut être renommée depuis sa page d'appareil (nécessite Home Assistant **2024.11+**, validé jusqu'à **2026.7**).
- **Commandes principales** (si accessible en écriture) : **marche/arrêt** de l'installation, **saison** (hiver/été) et **marche/arrêt ACS**.
- **Entités climate natives** (mode *Intégration avancée* facultatif) pour chaque zone active, avec `hvac_action` chauffage/refroidissement, une seule température cible (la consigne confort de la saison), des presets traduits (`eco`/`comfort`) et, si disponible, contrôle de l'humidité.
- **Familles d'appareils supplémentaires** exposées en diagnostic lecture seule : unités de pompe à chaleur et régulateur de cascade, cascade de générateurs OpenTherm (si activée), sorties relais de la carte, alarmes système, point de rosée de zone, températures de retour et pompes des circuits, températures des sources et totaux d'énergie sur 32 bits. Les canaux indisponibles restent *inconnus* au lieu d'afficher des valeurs erronées.
- **Renommage des entités depuis les réglages de l'add-on** : nommez une zone une fois et toutes ses entités suivent.
- **Suivi de disponibilité** via MQTT Last Will : les entités passent en `unavailable` si l'add-on s'arrête.
- **Auto-réparation** : reconnexion automatique à l'expiration de la session Setecna, reconnexion MQTT automatique avec backoff, republication de la découverte au redémarrage de Home Assistant, reconstruction automatique des entités climate au changement hiver/été.
- Prise en charge d'un **broker MQTT personnalisé** ou détection automatique de l'add-on Mosquitto.
- **Entité de mise à jour** qui indique la version en cours et signale les nouvelles versions GitHub.

## Installation

1. Assurez-vous d'utiliser Home Assistant **2024.11 ou plus récent** avec un broker MQTT (p. ex. l'add-on Mosquitto) et l'intégration MQTT configurée.
2. Cliquez sur le bouton ci-dessus, ou ajoutez `https://github.com/marturano/setecna-reg-plus` dans *Paramètres → Add-ons → Boutique d'add-ons → ⋮ → Dépôts*.
3. Installez **Setecna REG PLUS**, remplissez la configuration et démarrez-le.

## Configuration

| Option | Requis | Description |
|---|---|---|
| `systemID` | oui | Votre ID de système, affiché dans l'interface web s5a.eu une fois connecté |
| `username` | oui | L'e-mail de votre compte s5a.eu |
| `password` | oui | Le mot de passe de votre compte s5a.eu |
| `readonly` | non (`false`) | N'exposer que les capteurs ; ne jamais écrire vers le système |
| `adv_int` | non (`false`) | Créer des entités `climate` natives par zone (nécessite `readonly: false`) |
| `cleanup_legacy` | non (`true`) | Supprimer les topics de découverte par entité créés par la v1.x |
| `poll_interval` | non (`30`) | Secondes entre les rafraîchissements depuis le cloud Setecna (10–600) |
| `mqtt_host` | non | Hôte du broker MQTT personnalisé. Vide = détecter l'add-on Mosquitto |
| `mqtt_port` | non (`1883`) | Port du broker personnalisé (uniquement avec `mqtt_host`) |
| `mqtt_username` | non | Nom d'utilisateur du broker personnalisé (vide = anonyme) |
| `mqtt_password` | non | Mot de passe du broker personnalisé |
| `entity_names` | non | Renommage des entités, une entrée `PRÉFIXE=Nom` (voir ci-dessous) |
| `active_zones` | non | Numéros de zone à exposer ; vide = toutes les zones détectées |

### Renommage des entités

Ajoutez une entrée `PRÉFIXE=Nom` par ligne sous `entity_names` :

```
Z1=Salles de bain
Z3=Salon
C1=Circuit mélangé panneaux
GLOBAL_OUTPUT_3=Pompe de recirculation
```

Un **préfixe** de zone/circuit/source/pompe à chaleur (`Z1`, `C1`, `S1`, `HP0`) renomme d'un coup toutes les entités de cet élément et son thermostat ; un **identifiant de paramètre exact** renomme une seule entité. Au démarrage, l'add-on journalise les libellés personnalisés de votre panneau Setecna à copier.

Voir [`setecna/DOCS.md`](setecna/DOCS.md) (en anglais) pour les topics MQTT, les entités de diagnostic, les totaux des compteurs d'énergie, les notes de mise à niveau et les détails de résilience.

## Migration depuis l'add-on original

Si vous utilisiez auparavant l'add-on original `homeassistant-addon-setecna` (1.x) d'ingordigia, les `unique_id` des entités sont inchangés, donc les entités, l'historique et les tableaux de bord sont préservés. Les topics d'état bruts passent à `setecna/<systemID>/<PARAM>` ; avec `cleanup_legacy: true`, les anciens topics de découverte par entité sont supprimés au premier démarrage. Si vous n'avez jamais utilisé l'add-on original, installez-le simplement. Détails complets dans le [changelog](setecna/CHANGELOG.md).

---

## Crédits

Cet add-on est un fork du projet original [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) par **ingordigia**, réécrit en Go et largement étendu. Distribué sous licence Apache-2.0.

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
