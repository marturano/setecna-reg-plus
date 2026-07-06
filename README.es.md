# AVISO

Este add-on está desarrollado mediante ingeniería inversa de la interfaz web de Setecna y **no** cuenta con soporte oficial del equipo de Setecna. Úselo bajo su propia responsabilidad.

# Repositorio del add-on de Home Assistant para Setecna REG

> 🌐 **Idiomas:** 🇬🇧 [English](README.md) · 🇮🇹 [Italiano](README.it.md) · 🇩🇪 [Deutsch](README.de.md) · 🇫🇷 [Français](README.fr.md) · 🇪🇸 Español

Este repositorio contiene el add-on de Home Assistant que integra un sistema térmico **Setecna REG** en Home Assistant mediante MQTT, usando el [descubrimiento basado en dispositivos](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery).

[![Abra su instancia de Home Assistant y muestre el diálogo para añadir un repositorio de add-ons.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fmarturano%2Fsetecna-reg-plus)

![Compatible con arquitectura aarch64][aarch64-shield]
![Compatible con arquitectura amd64][amd64-shield]
![Compatible con arquitectura armv7][armv7-shield]
![Compatible con arquitectura armhf][armhf-shield]
![Compatible con arquitectura i386][i386-shield]

---

## Características

- **Un dispositivo de Home Assistant por elemento** (dispositivo principal *Setecna REG* más uno por cada zona, circuito, fuente, bomba de calor y ACS): las entidades quedan agrupadas y puede renombrar una zona entera desde su página de dispositivo (requiere Home Assistant **2024.11+**, verificado hasta **2026.7**).
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

Un **prefijo** de zona/circuito/fuente/bomba de calor (`Z1`, `C1`, `S1`, `HP0`) renombra de una vez todas las entidades de ese elemento y su termostato; un **id de parámetro exacto** renombra una sola entidad. Al iniciar, el add-on registra en el log las etiquetas personalizadas de su panel Setecna para copiarlas.

Consulte [`setecna/DOCS.md`](setecna/DOCS.md) (en inglés) para los topics MQTT, las entidades de diagnóstico, los totales de los contadores de energía, las notas de actualización y los detalles de resiliencia.

## Migración desde el add-on original

Si antes usaba el add-on original `homeassistant-addon-setecna` (1.x) de ingordigia, los `unique_id` de las entidades no cambian, por lo que se conservan las entidades, el historial y los paneles. Los topics de estado sin procesar pasan a `setecna/<systemID>/<PARAM>`; con `cleanup_legacy: true` los topics de descubrimiento por entidad antiguos se eliminan en el primer inicio. Si nunca usó el add-on original, solo instálelo. Detalles completos en el [changelog](setecna/CHANGELOG.md).

---

## Créditos

Este add-on es un fork del proyecto original [homeassistant-addon-setecna](https://github.com/Ingordigia/homeassistant-addon-setecna) de **ingordigia**, reescrito en Go y ampliamente extendido. Distribuido bajo la licencia Apache-2.0.

[aarch64-shield]: https://img.shields.io/badge/aarch64-yes-green.svg
[amd64-shield]: https://img.shields.io/badge/amd64-yes-green.svg
[armv7-shield]: https://img.shields.io/badge/armv7-yes-green.svg
[armhf-shield]: https://img.shields.io/badge/armhf-yes-green.svg
[i386-shield]: https://img.shields.io/badge/i386-yes-green.svg
