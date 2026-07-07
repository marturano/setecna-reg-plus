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

## Características

- **Un dispositivo de Home Assistant por zona** (más el dispositivo principal *Setecna REG* que agrupa globales, ACS, circuitos, fuentes, bombas de calor y controlador): puede renombrar una zona entera desde su página de dispositivo (requiere Home Assistant **2024.11+**, verificado hasta **2026.7**).
- **Controles principales** (cuando es escribible): **encendido/apagado** de la instalación, **temporada** (invierno/verano) y **encendido/apagado ACS**.
- **Entidades climate nativas** (modo *Integración avanzada* opcional) para cada zona activa, con `hvac_action` de calefacción/refrigeración, una única temperatura objetivo (la consigna confort de la temporada), presets traducidos (`eco`/`comfort`) y, cuando esté disponible, control de humedad.
- **Familias de dispositivos adicionales** expuestas como diagnóstico de solo lectura: unidades de bomba de calor y controlador de cascada, cascada de generadores OpenTherm (cuando está habilitada), salidas de relé de la placa, alarmas del sistema, punto de rocío de zona, temperaturas de retorno y bombas de los circuitos, temperaturas de las fuentes y totales de energía de 32 bits. Los canales no disponibles permanecen como *desconocido* en lugar de mostrar valores erróneos.
- **Renombrado de entidades desde los ajustes del add-on**: nombre una zona una vez y todas sus entidades lo heredan.
- **Seguimiento de disponibilidad** mediante MQTT Last Will: las entidades pasan a `unavailable` si el add-on se detiene.
- **Autorreparación**: reinicio de sesión automático cuando expira la sesión de Setecna, reconexión MQTT automática con backoff, republicación del descubrimiento al reiniciar Home Assistant, reconstrucción automática de las entidades climate al cambiar invierno/verano.
- Compatibilidad con un **broker MQTT personalizado** o detección automática del add-on Mosquitto.
- **Entidad de actualización** que informa de la versión en ejecución y destaca las nuevas versiones de GitHub.

## Instalación

1. Asegúrese de usar Home Assistant **2024.11 o posterior** con un broker MQTT (p. ej. el add-on Mosquitto) y la integración MQTT configurada.
2. Pulse el botón de arriba, o añada `https://github.com/marturano/setecna-reg-plus` en *Ajustes → Add-ons → Tienda de add-ons → ⋮ → Repositorios*.
3. Instale **Setecna REG PLUS**, rellene la configuración e inícielo.

## Configuración

| Opción | Obligatoria | Descripción |
|---|---|---|
| `systemID` | sí | Su ID de sistema, visible en la interfaz web s5a.eu tras iniciar sesión |
| `username` | sí | El correo electrónico de su cuenta de s5a.eu |
| `password` | sí | La contraseña de su cuenta de s5a.eu |
| `readonly` | no (`false`) | Exponer solo los sensores; nunca escribir en el sistema |
| `adv_int` | no (`false`) | Crear entidades `climate` nativas por zona (requiere `readonly: false`) |
| `cleanup_legacy` | no (`true`) | Eliminar los topics de descubrimiento por entidad creados por la v1.x |
| `poll_interval` | no (`30`) | Segundos entre actualizaciones desde la nube de Setecna (10–600) |
| `mqtt_host` | no | Host del broker MQTT personalizado. Vacío = detectar el add-on Mosquitto |
| `mqtt_port` | no (`1883`) | Puerto del broker personalizado (solo con `mqtt_host`) |
| `mqtt_username` | no | Usuario del broker personalizado (vacío = anónimo) |
| `mqtt_password` | no | Contraseña del broker personalizado |
| `entity_names` | no | Renombrado de entidades, una entrada `PREFIJO=Nombre` (ver abajo) |
| `active_zones` | no | Números de zona a exponer; vacío = todas las zonas detectadas |

### Renombrado de entidades

Añada una entrada `PREFIJO=Nombre` por línea en `entity_names`:

```
Z1=Baños
Z3=Salón
C1=Circuito mezclado paneles
GLOBAL_OUTPUT_3=Bomba de recirculación
```

Un **prefijo de zona** (`Z1`, `Z2`, ...) renombra el dispositivo de esa zona y, con él, todas las entidades de la zona y su termostato de una vez; un **id de parámetro exacto** (p. ej. `GLOBAL_OUTPUT_3`) renombra una sola entidad en el dispositivo principal. Los prefijos e ids **distinguen mayúsculas y minúsculas** (`Z1`, no `z1`). Al iniciar, el add-on registra en el log las etiquetas personalizadas de su panel Setecna para copiarlas.

Consulte [`setecna/DOCS.md`](setecna/DOCS.md) (en inglés) para los topics MQTT, las entidades de diagnóstico, los totales de los contadores de energía, las notas de actualización y los detalles de resiliencia.

## Migración desde el add-on original

Si antes usaba el add-on original `homeassistant-addon-setecna` (1.x) de ingordigia, los `unique_id` de las entidades no cambian, por lo que se conservan las entidades, el historial y los paneles. Los topics de estado sin procesar pasan a `setecna/<systemID>/<PARAM>`; con `cleanup_legacy: true` los topics de descubrimiento por entidad antiguos se eliminan en el primer inicio. Si nunca usó el add-on original, solo instálelo. Detalles completos en el [changelog](setecna/CHANGELOG.md).

---

## Créditos

Este add-on es un fork del proyecto original [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) de **ingordigia**, reescrito en Go y ampliamente extendido. Distribuido bajo la licencia Apache-2.0.

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
