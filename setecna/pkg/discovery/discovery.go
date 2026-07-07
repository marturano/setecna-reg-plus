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
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Ingordigia/homeassistant-addon-setecna/models"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/helpers"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/mqtt"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/scraper"
)

const (
	// Version of the add-on, shown as origin/software version in HA.
	Version = "1.0.2"

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
	// ActiveZones, when non-nil, is an allowlist of zone numbers to expose.
	// Zones detected on the panel but not in this set (and all their
	// entities and thermostat) are excluded and, if previously published,
	// removed. nil means "expose every detected zone" (default behaviour).
	ActiveZones map[int]bool
	// Diagnostics, when false (the default), removes all diagnostic entities
	// (raw codes, alarms, outputs, ...) instead of publishing them, keeping
	// the device pages clean. Set true to expose them (created disabled, so
	// the user can enable individual ones).
	Diagnostics bool
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

// nameOr returns the user override for key, or def when none is set.
func (b *Bridge) nameOr(key, def string) string {
	if b.Names != nil {
		if n, ok := b.Names[key]; ok && n != "" {
			return n
		}
	}
	return def
}

// elementOf returns the zone a parameter belongs to ("Z1", "Z2", ...) so each
// zone becomes its own Home Assistant device. Every other parameter (globals,
// ACS, circuits, sources, heat pumps, controller, meters, ...) returns "" and
// stays on the main device.
func elementOf(key string) string {
	m := leadRe.FindStringSubmatch(key)
	if m == nil || m[1] != "Z" {
		return ""
	}
	return "Z" + m[2]
}

// elementLead returns the default English lead of a zone ("Zone 1", ...), used
// both as the default device name and as the prefix stripped from entity labels.
func elementLead(elem string) string {
	m := leadRe.FindStringSubmatch(elem)
	if m == nil {
		return elem
	}
	return leadWords[m[1]] + " " + m[2]
}

// elementModel returns the device model shown in Home Assistant.
func elementModel(elem string) string {
	m := leadRe.FindStringSubmatch(elem)
	if m == nil {
		return ""
	}
	return leadWords[m[1]]
}

// deviceName returns the (possibly user-overridden) name of an element device.
func (b *Bridge) deviceName(elem string) string {
	return b.nameOr(elem, elementLead(elem))
}

// entityLabel returns the entity-specific part of a name. For element entities
// it strips the leading "<Element> " so Home Assistant composes
// "<device name> <label>"; for main-device entities it returns the full name.
// An exact parameter-id override always wins.
func (b *Bridge) entityLabel(key string, attr models.Attributes, elem string) string {
	if b.Names != nil {
		if n, ok := b.Names[key]; ok && n != "" {
			return n
		}
	}
	if elem == "" {
		return attr.Name
	}
	lead := elementLead(elem)
	label := attr.Name
	if strings.HasPrefix(attr.Name, lead+" ") {
		label = strings.TrimSpace(attr.Name[len(lead):])
	}
	return capitalize(label)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// zoneOf returns the zone number for a zone parameter id ("Z7_TEMP" -> 7),
// or 0 if the key is not a zone parameter.
func zoneOf(key string) int {
	m := leadRe.FindStringSubmatch(key)
	if m == nil || m[1] != "Z" {
		return 0
	}
	n, _ := strconv.Atoi(m[2])
	return n
}

// zoneExcluded reports whether the given zone number must be hidden given the
// ActiveZones allowlist. A nil allowlist means every zone is exposed.
func (b *Bridge) zoneExcluded(zone int) bool {
	return zone != 0 && b.ActiveZones != nil && !b.ActiveZones[zone]
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
// configTopicFor returns the device-based discovery config topic for a device
// identifier.
func (b *Bridge) configTopicFor(identifier string) string {
	return discoveryPrefix + "/device/setecna_" + identifier + "/config"
}

// ConfigTopic is the discovery topic of the main device.
func (b *Bridge) ConfigTopic() string {
	return b.configTopicFor(b.SystemID)
}

// deviceGroup accumulates the components belonging to one Home Assistant device.
type deviceGroup struct {
	identifier string
	name       string
	model      string
	main       bool
	components map[string]any
}

// DeviceConfigs builds the discovery payloads: one for the main "Setecna REG"
// device (system-level entities) and one per active element (each zone,
// circuit, source, heat pump, ...) linked to the main device via via_device.
// Splitting into sub-devices lets Home Assistant compose "<device> <label>"
// names and lets the user rename a whole zone from the device page.
//
// params holds the enabled entities, responseMap the last full snapshot
// (used to detect active zones / humidity), advClimate enables native climate
// entities. It also returns removal messages for excluded zones' sub-devices.
func (b *Bridge) DeviceConfigs(params models.ParamsMap, responseMap map[string]string, advClimate bool) ([]mqtt.Message, error) {
	groups := map[string]*deviceGroup{}
	var order []string
	group := func(elem string) *deviceGroup {
		g, ok := groups[elem]
		if ok {
			return g
		}
		g = &deviceGroup{components: map[string]any{}}
		if elem == "" {
			g.identifier, g.name, g.model, g.main = b.SystemID, "Setecna REG", "REG system", true
		} else {
			g.identifier = b.SystemID + "_" + elem
			g.name = b.deviceName(elem)
			g.model = elementModel(elem)
		}
		groups[elem] = g
		order = append(order, elem)
		return g
	}
	group("") // the main device always exists

	for key, attr := range params {
		if b.zoneExcluded(zoneOf(key)) {
			continue // excluded zones are removed as whole sub-devices below
		}
		cmp := b.component(key, attr, b.entityLabel(key, attr, elementOf(key)))
		if cmp == nil {
			continue
		}
		// When diagnostics are disabled, publish an empty config for diagnostic
		// entities so Home Assistant removes any previously-created ones and
		// does not re-create them.
		if !b.Diagnostics && cmp["entity_category"] == "diagnostic" {
			cmp = map[string]any{"platform": attr.EntityType}
		}
		group(elementOf(key)).components[key] = cmp
	}

	// Self-update entity lives on the main device.
	group("").components["addon_update"] = b.updateComponent()

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
			if b.zoneExcluded(i) {
				continue
			}
			withHumidity := responseMap[zone+"_RH"] != "32769" && responseMap[zone+"_RH"] != ""
			group(zone).components[fmt.Sprintf("zone_%d", i)] = b.climateComponent(i, season, withHumidity)
		}
	}

	var msgs []mqtt.Message
	for _, elem := range order {
		g := groups[elem]
		dev := map[string]any{
			"identifiers":  []string{g.identifier},
			"name":         g.name,
			"manufacturer": "Setecna",
			"model":        g.model,
		}
		if g.main {
			dev["sw_version"] = Version
		} else {
			dev["via_device"] = b.SystemID
		}
		payload := map[string]any{
			"device": dev,
			"origin": map[string]any{
				"name":        "Setecna REG PLUS",
				"sw_version":  Version,
				"support_url": repoURL,
			},
			"availability_topic": b.AvailabilityTopic(),
			"qos":                0,
			"components":         g.components,
		}
		j, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshalling device %q discovery: %w", g.identifier, err)
		}
		msgs = append(msgs, mqtt.Message{
			Topic:   b.configTopicFor(g.identifier),
			Payload: string(j),
			Qos:     1,
			Retain:  true,
		})
	}

	// Remove sub-devices for zones that are detected but excluded by the
	// active_zones allowlist (empty retained payload deletes the device).
	for i := 1; i <= 32; i++ {
		zone := fmt.Sprintf("Z%d", i)
		if responseMap[zone+"_SENSOR_CHN"] == "0" || responseMap[zone+"_SENSOR_CHN"] == "" {
			continue
		}
		if b.zoneExcluded(i) {
			msgs = append(msgs, mqtt.Message{
				Topic:  b.configTopicFor(b.SystemID + "_" + zone),
				Qos:    1,
				Retain: true,
			})
		}
	}
	return msgs, nil
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

// setControlCategory sets the entity_category for a writable control. An empty
// category defaults to "config" (tucked into the device's configuration
// section); the sentinel "primary" leaves it unset so the control appears as a
// main entity (and is exposed prominently to voice assistants).
func setControlCategory(base map[string]any, category string) {
	if category == "" {
		category = "config"
	}
	if category != "primary" {
		base["entity_category"] = category
	}
}

// component maps a Setecna parameter to a discovery component config. name is
// the entity label (already stripped of the element prefix by the caller).
func (b *Bridge) component(key string, attr models.Attributes, name string) map[string]any {
	base := map[string]any{
		"platform":  attr.EntityType,
		"unique_id": b.SystemID + "_" + key,
		"name":      name,
	}
	addIf := func(k, v string) {
		if v != "" {
			base[k] = v
		}
	}
	// Command param may differ from the read key (see Attributes.WriteKey).
	cmdKey := key
	if attr.WriteKey != "" {
		cmdKey = attr.WriteKey
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
		// Diagnostic entities are created but disabled by default to keep the
		// device page uncluttered; the user can enable the ones they want.
		if base["entity_category"] == "diagnostic" {
			base["enabled_by_default"] = false
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
		if base["entity_category"] == "diagnostic" {
			base["enabled_by_default"] = false
		}
		base["payload_on"] = "on"
		base["payload_off"] = "off"
		addIf("device_class", attr.DeviceClass)
		addIf("value_template", attr.ValueTemplate)
	case "number":
		base["state_topic"] = b.StateTopic(key)
		base["command_topic"] = b.CommandTopic(cmdKey)
		base["command_template"] = "{{ (value * 10) | int }}"
		setControlCategory(base, attr.EntityCategory)
		base["mode"] = "slider"
		base["min"] = attr.Min
		base["max"] = attr.Max
		base["step"] = attr.Step
		addIf("device_class", attr.DeviceClass)
		addIf("unit_of_measurement", attr.UnitOfMeasurement)
		addIf("value_template", attr.ValueTemplate)
	case "switch":
		base["state_topic"] = b.StateTopic(key)
		base["command_topic"] = b.CommandTopic(cmdKey)
		base["payload_on"] = "1"
		base["payload_off"] = "0"
		setControlCategory(base, attr.EntityCategory)
		addIf("device_class", attr.DeviceClass)
		addIf("value_template", attr.ValueTemplate)
	case "select":
		base["state_topic"] = b.StateTopic(key)
		base["command_topic"] = b.CommandTopic(cmdKey)
		setControlCategory(base, attr.EntityCategory)
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
		// A normal entity label (not the device's main entity): Home Assistant
		// shows it as "<zone> Thermostat" and it decouples cleanly from the
		// device name, so renaming/regenerating IDs behaves like any other
		// entity.
		"name": "Thermostat",

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

		// No preset_modes: Amazon Alexa special-cases the "eco" preset and
		// mis-reports the thermostat state (shows "eco" even when it is not).
		// The economy/comfort forcing is exposed as a separate select instead.
	}

	// Single target temperature mapped to the comfort setpoint of the active
	// season (CW winter / CS summer). A single setpoint - instead of a
	// low/high range - is what Alexa (and most UIs) expect for a heat-only or
	// cool-only thermostat: a range in a single-mode climate makes Alexa hang
	// on load. The economy setpoint stays adjustable as its own number entity
	// (Z*_SET_EW / Z*_SET_ES).
	comfort := z + "_SET_CW"
	if season == helpers.Summer {
		comfort = z + "_SET_CS"
	}
	c["temperature_state_topic"] = b.StateTopic(comfort)
	c["temperature_state_template"] = scaleDown
	c["temperature_command_topic"] = b.CommandTopic(comfort)
	c["temperature_command_template"] = scaleUp

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
		if b.zoneExcluded(zoneOf(d.ID)) {
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
