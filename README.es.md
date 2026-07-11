# AVISO

Esta es una integración no oficial, desarrollada por la comunidad. **No está afiliada, respaldada, patrocinada ni cuenta con soporte de ninguna manera por parte de SETECNA EPC Srl**, proveedora del sistema Setecna REG y del portal de telegestión s5a.eu (www.s5a.eu).

«Setecna», «Setecna REG», «s5a.eu» y cualquier nombre o logotipo relacionado son marcas o propiedad de sus respectivos titulares, utilizados aquí únicamente con fines de identificación e interoperabilidad. Este proyecto se desarrolla de forma independiente, mediante ingeniería inversa de la interfaz web de s5a.eu accesible públicamente.

**Sin garantía.** El software se proporciona «tal cual», sin garantía de ningún tipo, expresa o implícita, incluidas la comerciabilidad, la idoneidad para un fin determinado, la exactitud o la no infracción.

**Limitación de responsabilidad.** En la máxima medida permitida por la ley, el autor o los autores no asumen responsabilidad alguna por daños, pérdidas o costes de cualquier tipo - directos, indirectos, incidentales, especiales o consecuentes - derivados del uso o de la imposibilidad de usar esta integración. Esto incluye, sin limitación: mal funcionamiento, daño o desgaste de la instalación de calefacción/refrigeración o de sus componentes; lecturas o comandos erróneos, ausentes o retardados; pérdida de confort, desperdicio de energía o mayores costes; pérdida de datos; interrupción o suspensión del servicio s5a.eu o de su cuenta.

**Uso bajo su propia responsabilidad.** La integración puede leer y, si se habilita, **escribir** parámetros en su sistema Setecna REG (consignas, temporada, encendido/apagado, forzados, ...). Dichas escrituras modifican el comportamiento real de su instalación. Usted es el único responsable del uso que haga y de cualquier cambio que produzca en su sistema. Esta integración **no es un dispositivo de seguridad** y no debe utilizarse para ninguna función crítica de seguridad.

**Condiciones de terceros.** Es su responsabilidad asegurarse de que su uso cumpla los términos de servicio de SETECNA EPC Srl relativos al sistema Setecna REG y al portal s5a.eu. El proveedor puede cambiar o retirar el acceso en cualquier momento, lo que puede impedir el funcionamiento de esta integración.

# Repositorio del add-on de Home Assistant para Setecna REG

> 🌐 **Idiomas:** 🇬🇧 [English](README.md) · 🇮🇹 [Italiano](README.it.md) · 🇩🇪 [Deutsch](README.de.md) · 🇫🇷 [Français](README.fr.md) · 🇪🇸 Español

Este repositorio contiene el add-on de Home Assistant que integra un sistema térmico **Setecna REG** en Home Assistant mediante MQTT, usando el [descubrimiento basado en dispositivos](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery).

[![Abra su instancia de Home Assistant y muestre el diálogo para añadir un repositorio de add-ons.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fmarturano%2Fsetecna-reg-plus)

![Compatible con arquitectura aarch64][aarch64-shield]
![Compatible con arquitectura amd64][amd64-shield]

---

## Como funciona

Los sistemas Setecna REG se gestionan a traves del portal cloud **s5a.eu**. El add-on inicia sesion en el portal, obtiene a intervalos regulares la instantanea completa del sistema y la republica en Home Assistant por **MQTT** (descubrimiento basado en dispositivo); los comandos de Home Assistant se traducen de vuelta en escrituras en el portal.

```
Instalacion Setecna REG  <->  cloud s5a.eu  <->  [ add-on ]  <->  broker MQTT  <->  Home Assistant
```

La instalacion aparece como un **dispositivo principal** (sensores y controles del sistema) mas **un subdispositivo por zona activa**. Los `unique_id` de las entidades son estables, por lo que el historial, los paneles y los renombrados manuales se conservan entre reinicios y actualizaciones.

## Caracteristicas

- **Un dispositivo de Home Assistant por zona**, mas el dispositivo principal *Setecna REG* (globales, ACS, circuitos, fuentes, contadores de energia, calendarios y diagnosticos opcionales). Requiere Home Assistant **2024.11+**.
- **Termostato nativo** por zona (opcional): `heat`/`cool` de consigna unica + `off`, temperatura actual y humedad actual/ajustada (0-100%) si hay sonda. Optimizado para **Amazon Alexa**.
- Sensor **Regime** (automatico/forzado x confort/eco en tiempo real, correcto incluso con la zona en reposo) y selector **Forcing** para anular el programa.
- **Apagado se muestra cuando la zona esta realmente apagada** (forzada o por programa), no solo al forzarla a mano.
- **Calendarios por zona**, consignas estacionales confort/economia, punto de rocio, estado de zona.
- **Entidades de diagnostico** (opcionales): bombas de calor, controlador en cascada, cascada OpenTherm, salidas de rele, alarmas.
- **`hide_unavailable`** (activado por defecto): los canales no cableados/instalados se ocultan en lugar de aparecer como *desconocido*.
- **Renombrado de entidades**, **controles maestros** (sistema/temporada/ACS), **filtro de zonas**, **cinco idiomas** (en/it/de/fr/es).
- **Resiliente**: reinicio de sesion en el cloud, reconexion MQTT con disponibilidad (Last Will), republicacion al reiniciar HA, backoff ante errores, entidad de auto-actualizacion.

## Instalacion

1. Anade este repositorio a la tienda de add-ons e instala **Setecna REG PLUS**.
2. En la **Configuracion**, indica al menos `systemID`, `username` y `password` (cuenta s5a.eu).
3. Inicia el add-on. El dispositivo y sus zonas aparecen en **Ajustes -> Dispositivos y servicios -> MQTT**.

Se necesitan un broker MQTT (p. ej. *Mosquitto broker*) y la integracion MQTT habilitada. El broker se detecta automaticamente por el Supervisor, o se define manualmente con las opciones `mqtt_*`.

## Configuracion

Las opciones mas utiles (tabla completa y detalles en [`setecna/DOCS.md`](setecna/DOCS.md)):

| Opcion | Defecto | Descripcion |
|---|---|---|
| `systemID` / `username` / `password` | - | Credenciales s5a.eu **obligatorias**. |
| `readonly` | `false` | Solo sensores; ningun control. |
| `adv_int` | `false` | Termostato por zona (requiere `readonly: false`). |
| `diagnostics` | `false` | Bombas de calor / OpenTherm / reles / alarmas. |
| `hide_unavailable` | `true` | Oculta canales no disponibles en vez de *desconocido*. |
| `language` | `en` | `en` / `it` / `de` / `fr` / `es`. |
| `active_zones` | `[]` | Limita las zonas expuestas (vacio = todas). |
| `entity_names` | `[]` | Renombra entidades (`Z1=Bagni`, ...). |
| `poll_interval` | `30` | Segundos entre refrescos del cloud (10-600). |

Documentacion completa en **[`setecna/DOCS.md`](setecna/DOCS.md)**.

## Migracion desde el add-on original

Los `unique_id` no cambian, por lo que entidades, historial y paneles se conservan. En el primer arranque, `cleanup_legacy` elimina los antiguos topics de descubrimiento por entidad; si quedan entidades obsoletas, reinicia Home Assistant una vez.

## Créditos

Este add-on es un fork del proyecto original [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) de **ingordigia**, reescrito en Go y ampliamente extendido. Distribuido bajo la licencia Apache-2.0.

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
