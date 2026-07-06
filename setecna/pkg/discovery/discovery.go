// Package discovery builds Home Assistant MQTT device-based discovery
// payloads (introduced in HA 2024.11) for the Setecna REG system.
//
// A single retained message on homeassistant/device/<id>/config describes
// the device and all of its components, replacing the hundreds of
// per-entity config topics used by v1.x of this add-on.
package discovery

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/Ingordigia/homeassistant-addon-setecna/models"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/helpers"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/mqtt"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/scraper"
)

const (
	// Version of the add-on, shown as origin/software version in HA.
	Version = "1.0.0"

	discoveryPrefix = "homeassistant"
	// REBRAND: if you fork this under a different GitHub owner/repo name,
	// update this URL (and githubRepo in cmd/main.go) to match.
	repoURL = "https://github.com/marturano/setecna-reg-plus"
)

// Bridge translates between Setecna parameters and Home Assistant topics.
type Bridge struct {
	SystemID  string
	BaseTopic string // root of all state/command topics, e.g. "setecna/<systemID>"
	// Names holds user-provided friendly-name overrides, keyed either by
	// element prefix ("Z1", "C1", "HP0", ...) or by exact parameter id
	// ("GLOBAL_OUTPUT_3"). A prefix override renames every entity of that
	// element (e.g. "Zone 1 temperature" -> "Bagni temperature").
	Names map[string]string
}

// New creates a Bridge for the given system. names may be nil.
func New(systemID string, names map[string]string) *Bridge {
	return &Bridge{
		SystemID:  systemID,
		BaseTopic: "setecna/" + systemID,
		Names:     names,
	}
}

var (
	// primarySensor lists the device classes whose sensors should appear in
	// the main device view (and remain selectable in the Energy dashboard).
	primarySensor = map[string]bool{
		"temperature": true, "humidity": true, "power": true, "energy": true,
	}
	// leadRe extracts the element prefix (word + index) from a parameter id.
	// Longer prefixes come first because Go regexp is leftmost-first.
	leadRe    = regexp.MustCompile(`^(OT_G|FALDIN|FAIN|FDIN|HP|EM|MT|Z|C|S|D)(\d+)`)
	leadWords = map[string]string{
		"Z": "Zone", "C": "Circuit", "S": "Source", "HP": "Heat pump",
		"D": "Dehumidifier", "EM": "Energy meter", "FAIN": "Analog input",
		"FDIN": "Digital input", "FALDIN": "Alarm", "MT": "Calendar",
		"OT_G": "OpenTherm generator",
	}
)

// friendlyName applies user name overrides. key is the parameter id (or a
// synthetic "Z<n>" for climate), defaultName is the built-in English name.
// An exact-id override wins over a prefix override; a prefix override
// replaces the leading "<Word> <n>" so the role suffix is preserved.
func (b *Bridge) friendlyName(key, defaultName string) string {
	if b.Names == nil {
		return defaultName
	}
	if n, ok := b.Names[key]; ok && n != "" {
		return n
	}
	m := leadRe.FindStringSubmatch(key)
	if m == nil {
		return defaultName
	}
	prefix := m[1] + m[2] // e.g. "Z" + "1"
	n, ok := b.Names[prefix]
	if !ok || n == "" {
		return defaultName
	}
	lead := leadWords[m[1]] + " " + m[2] // e.g. "Zone 1"
	if defaultName == lead {
		return n
	}
	if strings.HasPrefix(defaultName, lead+" ") {
		return n + defaultName[len(lead):] // "Bagni" + " temperature"
	}
	return n
}

// AvailabilityTopic is where the add-on publishes online/offline.
func (b *Bridge) AvailabilityTopic() string { return b.BaseTopic + "/availability" }

// StateTopic returns the state topic for a parameter.
func (b *Bridge) StateTopic(param string) string { return b.BaseTopic + "/" + param }

// CommandTopic returns the command topic for a parameter.
func (b *Bridge) CommandTopic(param string) string { return b.BaseTopic + "/" + param + "/set" }

// CommandFilter is the wildcard subscription matching all command topics.
func (b *Bridge) CommandFilter() string { return b.BaseTopic + "/+/set" }

// ConfigTopic is the device-based discovery topic.
func (b *Bridge) ConfigTopic() string {
	return discoveryPrefix + "/device/setecna_" + b.SystemID + "/config"
}

// DeviceConfig builds the full device discovery payload.
//
// params holds the enabled entities, responseMap the last full snapshot of
// values (used to detect active zones / humidity support), advClimate
// enables native climate entities for the active zones.
func (b *Bridge) DeviceConfig(params models.ParamsMap, responseMap map[string]string, advClimate bool) (mqtt.Message, error) {
	components := map[string]any{}

	for key, attr := range params {
		if cmp := b.component(key, attr); cmp != nil {
			components[key] = cmp
		}
	}

	// Self-update entity: reports the running add-on version and, when a
	// newer GitHub release exists, offers the update in Home Assistant.
	components["addon_update"] = b.updateComponent()

	if advClimate {
		season := helpers.Winter
		if responseMap["GLOBAL_SEASON"] != "0" {
			season = helpers.Summer
		}
		for i := 1; i <= 32; i++ {
			zone := fmt.Sprintf("Z%d", i)
			if responseMap[zone+"_SENSOR_CHN"] == "0" || responseMap[zone+"_SENSOR_CHN"] == "" {
				continue
			}
			withHumidity := responseMap[zone+"_RH"] != "32769" && responseMap[zone+"_RH"] != ""
			components[fmt.Sprintf("zone_%d", i)] = b.climateComponent(i, season, withHumidity)
		}
	}

	payload := map[string]any{
		"device": map[string]any{
			"identifiers":  []string{b.SystemID},
			"name":         b.SystemID,
			"manufacturer": "Setecna",
			"model":        "REG system",
			"sw_version":   Version,
		},
		"origin": map[string]any{
			"name":        "Setecna REG PLUS",
			"sw_version":  Version,
			"support_url": repoURL,
		},
		"availability_topic": b.AvailabilityTopic(),
		"qos":                0,
		"components":         components,
	}

	j, err := json.Marshal(payload)
	if err != nil {
		return mqtt.Message{}, fmt.Errorf("marshalling device discovery payload: %w", err)
	}
	return mqtt.Message{
		Topic:   b.ConfigTopic(),
		Payload: string(j),
		Qos:     1,
		Retain:  true,
	}, nil
}

// updateComponent describes the add-on self-update entity.
func (b *Bridge) updateComponent() map[string]any {
	return map[string]any{
		"platform":        "update",
		"unique_id":       b.SystemID + "_addon_update",
		"name":            "Add-on",
		"state_topic":     b.StateTopic("addon_update"),
		"device_class":    "firmware",
		"entity_category": "config",
	}
}

// UpdateStateMessage builds the retained state for the update entity. When
// latest is empty it falls back to the running version (no update shown).
func (b *Bridge) UpdateStateMessage(latest, releaseURL string) mqtt.Message {
	if latest == "" {
		latest = Version
	}
	payload := map[string]string{
		"installed_version": Version,
		"latest_version":    latest,
	}
	if releaseURL != "" {
		payload["release_url"] = releaseURL
	}
	j, _ := json.Marshal(payload)
	return mqtt.Message{
		Topic:   b.StateTopic("addon_update"),
		Payload: string(j),
		Qos:     0,
		Retain:  true,
	}
}

// component maps a Setecna parameter to a discovery component config.
func (b *Bridge) component(key string, attr models.Attributes) map[string]any {
	base := map[string]any{
		"platform":  attr.EntityType,
		"unique_id": b.SystemID + "_" + key,
		"name":      b.friendlyName(key, attr.Name),
	}
	addIf := func(k, v string) {
		if v != "" {
			base[k] = v
		}
	}

	switch attr.EntityType {
	case "sensor":
		base["state_topic"] = b.StateTopic(key)
		// Primary measurements (temperature, humidity, power, energy) stay in
		// the main view and remain usable in the Energy dashboard; everything
		// else (status/enum/raw codes, timestamps) is diagnostic. An explicit
		// EntityCategory on the attribute always wins.
		if attr.EntityCategory != "" {
			base["entity_category"] = attr.EntityCategory
		} else if !primarySensor[attr.DeviceClass] {
			base["entity_category"] = "diagnostic"
		}
		addIf("device_class", attr.DeviceClass)
		addIf("value_template", attr.ValueTemplate)
		if attr.DeviceClass == "enum" {
			// enum sensors require an options list and must NOT carry
			// state_class or unit_of_measurement (Home Assistant rejects it).
			base["options"] = attr.Options
		} else {
			addIf("state_class", attr.StateClass)
			addIf("unit_of_measurement", attr.UnitOfMeasurement)
		}
	case "binary_sensor":
		base["state_topic"] = b.StateTopic(key)
		if attr.EntityCategory != "" {
			base["entity_category"] = attr.EntityCategory
		} else {
			base["entity_category"] = "diagnostic"
		}
		base["payload_on"] = "on"
		base["payload_off"] = "off"
		addIf("device_class", attr.DeviceClass)
		addIf("value_template", attr.ValueTemplate)
	case "number":
		base["state_topic"] = b.StateTopic(key)
		base["command_topic"] = b.CommandTopic(key)
		base["command_template"] = "{{ (value * 10) | int }}"
		base["entity_category"] = "config"
		base["mode"] = "slider"
		base["min"] = attr.Min
		base["max"] = attr.Max
		base["step"] = attr.Step
		addIf("device_class", attr.DeviceClass)
		addIf("unit_of_measurement", attr.UnitOfMeasurement)
		addIf("value_template", attr.ValueTemplate)
	case "select":
		base["state_topic"] = b.StateTopic(key)
		base["command_topic"] = b.CommandTopic(key)
		base["entity_category"] = "config"
		base["options"] = attr.Options
		addIf("command_template", attr.CommandTemplate)
		addIf("value_template", attr.ValueTemplate)
	default:
		slog.Warn("unsupported entity type, skipping", "param", key, "type", attr.EntityType)
		return nil
	}
	return base
}

// climateComponent builds the native climate entity for a zone.
func (b *Bridge) climateComponent(zone int, season helpers.Season, withHumidity bool) map[string]any {
	z := fmt.Sprintf("Z%d", zone)
	scaleDown := "{{ value | int / 10 }}"
	scaleUp := "{{ (value * 10) | int }}"

	onMode := "heat"
	action := "heating"
	if season == helpers.Summer {
		onMode = "cool"
		action = "cooling"
	}

	c := map[string]any{
		"platform":  "climate",
		"unique_id": fmt.Sprintf("%s_zone_%d", b.SystemID, zone),
		"name":      b.friendlyName(fmt.Sprintf("Z%d", zone), fmt.Sprintf("Zone %d", zone)),

		"current_temperature_topic":    b.StateTopic(z + "_TEMP"),
		"current_temperature_template": scaleDown,
		"temp_step":                    0.5,
		"min_temp":                     15,
		"max_temp":                     30,

		// The selected mode follows the zone forcing state: "forced off"
		// (1) means the zone is off, anything else means it is enabled.
		// Setting the mode writes the forcing: off -> 1, on -> 0 (automatic).
		"modes":            []string{onMode, "off"},
		"mode_state_topic": b.StateTopic(z + "_FORCING"),
		"mode_state_template": fmt.Sprintf(
			`{%% if value == "1" %%}off{%% else %%}%s{%% endif %%}`, onMode),
		"mode_command_topic": b.CommandTopic(z + "_FORCING"),
		"mode_command_template": fmt.Sprintf(
			`{%% if value == "%s" %%}0{%% else %%}1{%% endif %%}`, onMode),

		// hvac_action reflects whether the zone relay is actively calling
		// (heating/cooling) or idle, independent of the selected mode.
		"action_topic": b.StateTopic(z + "_OUTPUT"),
		"action_template": fmt.Sprintf(
			`{%% if value == "1" %%}%s{%% else %%}idle{%% endif %%}`, action),

		// Presets expose the finer forcing levels of the REG controller.
		"preset_modes":                 []string{"forced off", "forced economy", "forced comfort"},
		"preset_mode_command_topic":    b.CommandTopic(z + "_FORCING"),
		"preset_mode_command_template": `{% if value == "forced off" %}1{% elif value == "forced economy" %}2{% elif value == "forced comfort" %}3{% else %}0{% endif %}`,
		"preset_mode_state_topic":      b.StateTopic(z + "_FORCING"),
		"preset_mode_value_template":   `{% if value == "1" %}forced off{% elif value == "2" %}forced economy{% elif value == "3" %}forced comfort{% else %}none{% endif %}`,
	}

	// The REG controller exposes economy/comfort setpoints per season:
	// EW/CW (winter economy/comfort), ES/CS (summer economy/comfort).
	// The climate temperature_low/high pair maps to economy/comfort.
	if season == helpers.Summer {
		c["temperature_high_state_topic"] = b.StateTopic(z + "_SET_ES")
		c["temperature_high_command_topic"] = b.CommandTopic(z + "_SET_ES")
		c["temperature_low_state_topic"] = b.StateTopic(z + "_SET_CS")
		c["temperature_low_command_topic"] = b.CommandTopic(z + "_SET_CS")
	} else {
		c["temperature_low_state_topic"] = b.StateTopic(z + "_SET_EW")
		c["temperature_low_command_topic"] = b.CommandTopic(z + "_SET_EW")
		c["temperature_high_state_topic"] = b.StateTopic(z + "_SET_CW")
		c["temperature_high_command_topic"] = b.CommandTopic(z + "_SET_CW")
	}
	c["temperature_low_state_template"] = scaleDown
	c["temperature_low_command_template"] = scaleUp
	c["temperature_high_state_template"] = scaleDown
	c["temperature_high_command_template"] = scaleUp

	if withHumidity {
		c["current_humidity_topic"] = b.StateTopic(z + "_RH")
		c["current_humidity_template"] = scaleDown
		c["target_humidity_state_topic"] = b.StateTopic(z + "_SET_RH")
		c["target_humidity_state_template"] = scaleDown
		c["target_humidity_command_topic"] = b.CommandTopic(z + "_SET_RH")
		c["target_humidity_command_template"] = scaleUp
		c["min_humidity"] = 45
		c["max_humidity"] = 75
	}
	return c
}

// StateMessages converts a fetch response into retained state messages for
// the parameters that Home Assistant knows about.
func (b *Bridge) StateMessages(resp scraper.Response, params models.ParamsMap) mqtt.Messages {
	msgs := make(mqtt.Messages, 0, len(resp.Data))
	for _, d := range resp.Data {
		if _, ok := params[d.ID]; !ok {
			continue
		}
		payload := string(d.V)
		if d.ID == "LAST_UPDATE" {
			if payload == "" {
				continue
			}
			date, err := time.Parse("2006-01-02 15:04:05.000000-07", payload)
			if err != nil {
				slog.Debug("cannot parse LAST_UPDATE, ignoring", "value", payload, "error", err)
				continue
			}
			payload = date.Format(time.RFC3339)
		}
		msgs = append(msgs, mqtt.Message{
			Topic:   b.StateTopic(d.ID),
			Payload: payload,
			Qos:     0,
			Retain:  true,
		})
	}
	return msgs
}

// LegacyCleanupMessages returns empty retained payloads for all v1.x
// per-entity discovery topics, so upgrading users do not end up with
// duplicated or orphaned entities.
func (b *Bridge) LegacyCleanupMessages(params models.ParamsMap) mqtt.Messages {
	var msgs mqtt.Messages
	for key, attr := range params {
		msgs = append(msgs, mqtt.Message{
			Topic:  discoveryPrefix + "/" + attr.EntityType + "/" + b.SystemID + "_" + key + "/config",
			Qos:    0,
			Retain: true,
		})
	}
	for i := 1; i <= 32; i++ {
		msgs = append(msgs, mqtt.Message{
			Topic:  fmt.Sprintf("%s/climate/%s_zone_%d/config", discoveryPrefix, b.SystemID, i),
			Qos:    0,
			Retain: true,
		})
	}
	return msgs
}
