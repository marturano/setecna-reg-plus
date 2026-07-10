package discovery

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Ingordigia/homeassistant-addon-setecna/models"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/mqtt"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/scraper"
)

func testResponseMap() map[string]string {
	return map[string]string{
		"GLOBAL_SEASON": "0", // winter
		"Z1_SENSOR_CHN": "3", // active zone with humidity
		"Z1_RH":         "550",
		"Z2_SENSOR_CHN": "5", // active zone without humidity
		"Z2_RH":         "32769",
		"Z3_SENSOR_CHN": "0", // inactive zone
		"GLOBAL_T_EXT":  "123",
	}
}

// allComponents merges the components of every device message from
// DeviceConfigs, so tests can look up a component regardless of which
// sub-device it now lives on. Removal messages (empty payload) are skipped.
func allComponents(t *testing.T, b *Bridge, params models.ParamsMap, rm map[string]string, adv bool) map[string]map[string]any {
	t.Helper()
	msgs, err := b.DeviceConfigs(params, rm, adv)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]any{}
	for _, m := range msgs {
		if m.Payload == "" {
			continue
		}
		var p struct {
			Components map[string]map[string]any `json:"components"`
		}
		if err := json.Unmarshal([]byte(m.Payload), &p); err != nil {
			t.Fatalf("payload not JSON: %v", err)
		}
		for k, v := range p.Components {
			out[k] = v
		}
	}
	return out
}

// deviceOf returns the device block of the message that contains compKey.
func deviceOf(t *testing.T, b *Bridge, params models.ParamsMap, rm map[string]string, adv bool, compKey string) map[string]any {
	t.Helper()
	msgs, _ := b.DeviceConfigs(params, rm, adv)
	for _, m := range msgs {
		if m.Payload == "" {
			continue
		}
		var p struct {
			Device     map[string]any            `json:"device"`
			Components map[string]map[string]any `json:"components"`
		}
		json.Unmarshal([]byte(m.Payload), &p)
		if _, ok := p.Components[compKey]; ok {
			return p.Device
		}
	}
	return nil
}

func TestDeviceConfig(t *testing.T) {
	b := New("SYS1", nil)
	b.Diagnostics = true
	b.SystemControl = true
	b.SeasonControl = true
	b.ACSControl = true
	params := make(models.ParamsMap)
	params.AddEnabledParams(testResponseMap(), false)

	msgs, err := b.DeviceConfigs(params, testResponseMap(), true)
	if err != nil {
		t.Fatal(err)
	}
	// The main device carries origin + availability and lives at the main topic.
	var main mqtt.Message
	for _, m := range msgs {
		if m.Topic == "homeassistant/device/setecna_SYS1/config" {
			main = m
		}
	}
	if main.Topic == "" {
		t.Fatal("main device config missing")
	}
	if !main.Retain {
		t.Fatal("discovery payload must be retained")
	}
	var mainPayload struct {
		Device struct {
			Identifiers []string `json:"identifiers"`
		} `json:"device"`
		Origin struct {
			Name string `json:"name"`
		} `json:"origin"`
		AvailabilityTopic string `json:"availability_topic"`
	}
	if err := json.Unmarshal([]byte(main.Payload), &mainPayload); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if mainPayload.Origin.Name == "" {
		t.Fatal("origin is required by device-based discovery")
	}
	if mainPayload.AvailabilityTopic != "setecna/SYS1/availability" {
		t.Fatalf("wrong availability topic: %s", mainPayload.AvailabilityTopic)
	}

	components := allComponents(t, b, params, testResponseMap(), true)

	// Zone 1: climate with humidity, Zone 2: without, Zone 3: absent.
	z1, ok := components["zone_1"]
	if !ok {
		t.Fatal("zone_1 climate missing")
	}
	if z1["platform"] != "climate" {
		t.Fatalf("zone_1 platform = %v", z1["platform"])
	}
	if z1["unique_id"] != "SYS1_zone_1" {
		t.Fatalf("zone_1 unique_id = %v (must match v1 for seamless migration)", z1["unique_id"])
	}
	// The thermostat is a normal entity labelled "Thermostat" (not the
	// device's main entity), so HA composes "<zone> Thermostat".
	if z1["name"] != "Thermostat" {
		t.Fatalf("zone_1 climate name should be 'Thermostat', got %v", z1["name"])
	}
	if _, ok := z1["current_humidity_topic"]; !ok {
		t.Fatal("zone_1 should expose humidity")
	}
	z2 := components["zone_2"]
	if _, ok := z2["current_humidity_topic"]; ok {
		t.Fatal("zone_2 should NOT expose humidity")
	}
	if _, ok := components["zone_3"]; ok {
		t.Fatal("inactive zone_3 must not be discovered")
	}

	// zone_1 lives on its own sub-device linked to the main via via_device.
	dev := deviceOf(t, b, params, testResponseMap(), true, "zone_1")
	if dev == nil || dev["via_device"] != "SYS1" {
		t.Fatalf("zone_1 should be on a sub-device linked via_device to SYS1, got %v", dev)
	}
	if ids, _ := dev["identifiers"].([]any); len(ids) == 0 || ids[0] != "SYS1_Z1" {
		t.Fatalf("zone_1 sub-device identifier wrong: %v", dev["identifiers"])
	}

	// Winter: heat mode and single target = comfort (CW) setpoint.
	if !strings.Contains(z1["mode_state_template"].(string), "heat") {
		t.Fatal("winter climates must use heat mode")
	}
	if z1["temperature_state_topic"] != "setecna/SYS1/Z1_SET_CW" {
		t.Fatalf("winter target setpoint topic wrong: %v", z1["temperature_state_topic"])
	}
	// A single setpoint (no low/high range) is required for Alexa.
	if _, ok := z1["temperature_low_state_topic"]; ok {
		t.Fatal("climate must expose a single target, not a low/high range")
	}
	// Two presets so the field shows Auto/Eco instead of None: auto = automatic
	// (0), eco = forced eco (2), both wired to FORCING.
	if got := fmt.Sprintf("%v", z1["preset_modes"]); got != "[auto eco]" {
		t.Fatalf("climate preset_modes = %v", z1["preset_modes"])
	}
	if z1["preset_mode_state_topic"] != "setecna/SYS1/Z1_FORCING" {
		t.Fatalf("preset_mode_state_topic = %v", z1["preset_mode_state_topic"])
	}
	if z1["preset_mode_command_topic"] != "setecna/SYS1/Z1_FORCING/set" {
		t.Fatalf("preset_mode_command_topic = %v", z1["preset_mode_command_topic"])
	}

	// Every component must have platform + unique_id and state topics
	// outside the discovery prefix.
	for id, cmp := range components {
		if cmp["platform"] == nil || cmp["unique_id"] == nil {
			t.Fatalf("component %s missing platform/unique_id", id)
		}
		if st, ok := cmp["state_topic"].(string); ok && strings.HasPrefix(st, "homeassistant/") {
			t.Fatalf("component %s keeps state under the discovery prefix: %s", id, st)
		}
		if cmp["device_class"] == "enum" {
			if _, ok := cmp["options"]; !ok {
				t.Fatalf("enum component %s missing options", id)
			}
			if _, ok := cmp["state_class"]; ok {
				t.Fatalf("enum component %s must not set state_class", id)
			}
			if _, ok := cmp["unit_of_measurement"]; ok {
				t.Fatalf("enum component %s must not set unit_of_measurement", id)
			}
		}
	}
}

func TestStateMessages(t *testing.T) {
	b := New("SYS1", nil)
	params := make(models.ParamsMap)
	params.AddEnabledParams(testResponseMap(), true)

	resp := scraper.Response{Data: []scraper.Datum{
		{ID: "GLOBAL_T_EXT", V: "123"},
		{ID: "UNKNOWN_PARAM", V: "1"},
		{ID: "LAST_UPDATE", V: "2024-06-01 10:20:30.000000+02"},
	}}
	msgs := b.StateMessages(resp, params)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	for _, m := range msgs {
		if !m.Retain {
			t.Fatalf("state messages must be retained: %s", m.Topic)
		}
		if m.Topic == "setecna/SYS1/LAST_UPDATE" && !strings.Contains(m.Payload, "T") {
			t.Fatalf("LAST_UPDATE not converted to RFC3339: %s", m.Payload)
		}
	}
}

func TestSeasonSummer(t *testing.T) {
	b := New("SYS1", nil)
	rm := testResponseMap()
	rm["GLOBAL_SEASON"] = "1"
	z1 := allComponents(t, b, models.ParamsMap{}, rm, true)["zone_1"]
	if !strings.Contains(z1["action_template"].(string), "cooling") {
		t.Fatal("summer climates must report cooling action")
	}
	if z1["temperature_state_topic"] != "setecna/SYS1/Z1_SET_CS" {
		t.Fatalf("summer target setpoint topic wrong: %v", z1["temperature_state_topic"])
	}
}

func TestClimateModeFromForcing(t *testing.T) {
	b := New("SYS1", nil)
	z1 := allComponents(t, b, models.ParamsMap{}, testResponseMap(), true)["zone_1"]

	// Mode must be driven by FORCING, not by the relay output.
	if z1["mode_state_topic"] != "setecna/SYS1/Z1_FORCING" {
		t.Fatalf("mode should read FORCING, got %v", z1["mode_state_topic"])
	}
	// FORCING == 1 (forced off) => off, otherwise heat (winter).
	tmpl := z1["mode_state_template"].(string)
	if !strings.Contains(tmpl, `"1"`) || !strings.Contains(tmpl, "off") || !strings.Contains(tmpl, "heat") {
		t.Fatalf("mode template not mapping forced-off correctly: %s", tmpl)
	}
	// Action must come from the relay output.
	if z1["action_topic"] != "setecna/SYS1/Z1_OUTPUT" {
		t.Fatalf("action should read OUTPUT, got %v", z1["action_topic"])
	}
	if z1["mode_command_topic"] != "setecna/SYS1/Z1_FORCING/set" {
		t.Fatalf("mode command should write FORCING, got %v", z1["mode_command_topic"])
	}
}

func TestUpdateEntity(t *testing.T) {
	b := New("SYS1", nil)
	up, ok := allComponents(t, b, models.ParamsMap{}, testResponseMap(), false)["addon_update"]
	if !ok {
		t.Fatal("update entity missing from discovery")
	}
	if up["platform"] != "update" {
		t.Fatalf("addon_update platform = %v", up["platform"])
	}

	// State message must be valid JSON with installed/latest versions.
	sm := b.UpdateStateMessage("2.0.1", "https://example/releases/v2.0.1")
	var st struct {
		Installed  string `json:"installed_version"`
		Latest     string `json:"latest_version"`
		ReleaseURL string `json:"release_url"`
	}
	if err := json.Unmarshal([]byte(sm.Payload), &st); err != nil {
		t.Fatalf("update state not valid JSON: %v", err)
	}
	if st.Installed != Version || st.Latest != "2.0.1" {
		t.Fatalf("update versions wrong: %+v", st)
	}

	// Empty latest must fall back to the running version (no update shown).
	fallback := b.UpdateStateMessage("", "")
	if !strings.Contains(fallback.Payload, Version) {
		t.Fatalf("empty latest should fall back to installed version: %s", fallback.Payload)
	}
}

func TestNaming(t *testing.T) {
	names := map[string]string{
		"Z1":              "Bagni",
		"GLOBAL_OUTPUT_3": "Recirculation pump",
	}
	b := New("SYS1", names)
	b.Diagnostics = true

	// Element device name comes from the override; entity label is the
	// stripped, capitalised remainder.
	if got := b.deviceName("Z1"); got != "Bagni" {
		t.Fatalf("zone device name = %q", got)
	}
	if got := b.deviceName("Z2"); got != "Zone 2" {
		t.Fatalf("un-overridden zone device name = %q", got)
	}
	if got := b.entityLabel("Z1_TEMP", models.Attributes{Name: "Zone 1 temperature"}, "Z1"); got != "Temperature" {
		t.Fatalf("entity label = %q", got)
	}
	if got := b.entityLabel("Z1_DEWPOINT", models.Attributes{Name: "Zone 1 dew point"}, "Z1"); got != "Dew point" {
		t.Fatalf("entity label (dew point) = %q", got)
	}
	// Non-zone entities stay on the main device with their full name.
	if got := b.entityLabel("HPC_PID_TEMP", models.Attributes{Name: "Heat pump controller PID temperature"}, ""); got != "Heat pump controller PID temperature" {
		t.Fatalf("HPC entity on main device should keep full name, got %q", got)
	}
	// Main-device entity keeps its full name; exact-id override wins.
	if got := b.entityLabel("GLOBAL_OUTPUT_3", models.Attributes{Name: "Output 3"}, ""); got != "Recirculation pump" {
		t.Fatalf("exact override = %q", got)
	}
	if got := b.entityLabel("ANY_ALARM", models.Attributes{Name: "Any alarm"}, ""); got != "Any alarm" {
		t.Fatalf("main entity label = %q", got)
	}

	// In the real payload: the zone device is named "Bagni" and the sensor
	// label is "Temperature" (composed by HA as "Bagni Temperature").
	params := models.ParamsMap{"Z1_TEMP": models.Attributes{EntityType: "sensor", Name: "Zone 1 temperature"}}
	dev := deviceOf(t, b, params, map[string]string{}, false, "Z1_TEMP")
	if dev == nil || dev["name"] != "Bagni" {
		t.Fatalf("zone device should be named Bagni, got %v", dev)
	}
	if allComponents(t, b, params, map[string]string{}, false)["Z1_TEMP"]["name"] != "Temperature" {
		t.Fatal("zone sensor label should be 'Temperature'")
	}
}

func TestEntityCategoryRule(t *testing.T) {
	b := New("SYS1", nil)
	// Primary measurements must NOT be diagnostic (stay in main view / usable
	// in the Energy dashboard).
	for _, dc := range []string{"temperature", "humidity", "power", "energy"} {
		c := b.component("X", models.Attributes{EntityType: "sensor", DeviceClass: dc, Name: "x"}, "x")
		if _, ok := c["entity_category"]; ok {
			t.Fatalf("%s sensor must not be diagnostic", dc)
		}
	}
	// Raw code sensors (no device class) are diagnostic.
	raw := b.component("HP0_POWER", models.Attributes{EntityType: "sensor", Name: "raw"}, "raw")
	if raw["entity_category"] != "diagnostic" {
		t.Fatal("raw sensor must be diagnostic")
	}
	// Explicit override always wins.
	ov := b.component("X", models.Attributes{EntityType: "sensor", DeviceClass: "temperature", EntityCategory: "diagnostic", Name: "x"}, "x")
	if ov["entity_category"] != "diagnostic" {
		t.Fatal("explicit entity_category must win")
	}
}

func TestActiveZonesFilter(t *testing.T) {
	b := New("SYS1", nil)
	b.Diagnostics = true
	b.ActiveZones = map[int]bool{1: true, 2: true} // solo Z1 e Z2

	params := models.ParamsMap{
		"Z1_TEMP": models.Attributes{EntityType: "sensor", Name: "Zone 1 temperature"},
		"Z7_TEMP": models.Attributes{EntityType: "sensor", Name: "Zone 7 temperature"},
		"C1_TEMP": models.Attributes{EntityType: "sensor", Name: "Circuit 1 temperature"},
	}
	rm := map[string]string{
		"Z1_SENSOR_CHN": "57088",
		"Z7_SENSOR_CHN": "57088",
	}
	msgs, err := b.DeviceConfigs(params, rm, true)
	if err != nil {
		t.Fatal(err)
	}
	components := allComponents(t, b, params, rm, true)

	// Z1 esposta come sensore completo, C1 (non-zona) intatta.
	if components["Z1_TEMP"]["unique_id"] == nil {
		t.Fatal("Z1_TEMP should be a full component")
	}
	if components["C1_TEMP"]["unique_id"] == nil {
		t.Fatal("C1_TEMP (non-zone) should be unaffected")
	}
	if components["zone_1"] == nil {
		t.Fatal("zone_1 climate should be present")
	}
	// Z7 e il suo termostato non compaiono in alcun device.
	if _, ok := components["Z7_TEMP"]; ok {
		t.Fatal("excluded Z7_TEMP must not be published")
	}
	if _, ok := components["zone_7"]; ok {
		t.Fatal("excluded zone_7 climate must not be published")
	}
	// Il sotto-device della zona esclusa viene rimosso (payload vuoto).
	removed := false
	for _, m := range msgs {
		if m.Topic == b.configTopicFor("SYS1_Z7") && m.Payload == "" && m.Retain {
			removed = true
		}
	}
	if !removed {
		t.Fatal("excluded zone Z7 sub-device should be removed with an empty retained payload")
	}

	// Lo stato delle zone escluse non viene pubblicato.
	resp := scraper.Response{Data: []scraper.Datum{
		{ID: "Z1_TEMP", V: scraper.FlexString("258")},
		{ID: "Z7_TEMP", V: scraper.FlexString("260")},
	}}
	for _, m := range b.StateMessages(resp, params) {
		if m.Topic == b.StateTopic("Z7_TEMP") {
			t.Fatal("excluded zone Z7 should not publish state")
		}
	}
}

func TestActiveZonesNilExposesAll(t *testing.T) {
	b := New("SYS1", nil) // ActiveZones nil
	if b.zoneExcluded(7) {
		t.Fatal("with nil allowlist no zone should be excluded")
	}
	if zoneOf("Z7_TEMP") != 7 || zoneOf("C1_TEMP") != 0 {
		t.Fatal("zoneOf parsing wrong")
	}
}

func TestDiagnosticDisabledAndControls(t *testing.T) {
	b := New("SYS1", nil)

	// Diagnostic sensors are created but disabled by default.
	raw := b.component("HP0_STATUS", models.Attributes{EntityType: "sensor", Name: "s"}, "s")
	if raw["entity_category"] != "diagnostic" || raw["enabled_by_default"] != false {
		t.Fatalf("diagnostic sensor must be disabled by default, got %v", raw)
	}
	// Primary measurements stay enabled.
	temp := b.component("Z1_TEMP", models.Attributes{EntityType: "sensor", DeviceClass: "temperature", Name: "t"}, "t")
	if _, ok := temp["enabled_by_default"]; ok {
		t.Fatal("primary sensor must stay enabled")
	}
	// A "primary" control carries no entity_category (main control, not config).
	sw := b.component("GLOBAL_ENABLE", models.Attributes{EntityType: "switch", EntityCategory: "primary", Name: "System"}, "System")
	if _, ok := sw["entity_category"]; ok {
		t.Fatal("primary control must not be config/diagnostic")
	}
	if sw["command_topic"] != "setecna/SYS1/GLOBAL_ENABLE/set" || sw["payload_on"] != "1" {
		t.Fatalf("switch wiring wrong: %v", sw)
	}

	// When writable (readonly=false) the plant on/off and season become controls.
	m := make(models.ParamsMap)
	m.AddEnabledParams(map[string]string{"GLOBAL_SEASON": "0", "GLOBAL_ENABLE": "1"}, false)
	if m["GLOBAL_ENABLE"].EntityType != "switch" {
		t.Fatalf("system on/off should be a switch when writable, got %q", m["GLOBAL_ENABLE"].EntityType)
	}
	if m["GLOBAL_SEASON"].EntityType != "select" {
		t.Fatalf("season should be a select when writable, got %q", m["GLOBAL_SEASON"].EntityType)
	}
	// When read-only they stay sensors.
	ro := make(models.ParamsMap)
	ro.AddEnabledParams(map[string]string{"GLOBAL_SEASON": "0", "GLOBAL_ENABLE": "1"}, true)
	if ro["GLOBAL_ENABLE"].EntityType != "binary_sensor" || ro["GLOBAL_SEASON"].EntityType != "sensor" {
		t.Fatalf("read-only globals should be sensors, got %q/%q", ro["GLOBAL_ENABLE"].EntityType, ro["GLOBAL_SEASON"].EntityType)
	}
}

func TestGlobalWriteKeyPrefix(t *testing.T) {
	b := New("SYS1", nil)
	// Season select: state reads GLOBAL_SEASON, command writes P_GLOBAL_SEASON.
	c := b.component("GLOBAL_SEASON", models.Attributes{
		EntityType: "select", Options: []string{"winter", "summer"},
		WriteKey: "P_GLOBAL_SEASON",
	}, "Season")
	if c["state_topic"] != "setecna/SYS1/GLOBAL_SEASON" {
		t.Fatalf("season state topic wrong: %v", c["state_topic"])
	}
	if c["command_topic"] != "setecna/SYS1/P_GLOBAL_SEASON/set" {
		t.Fatalf("season command topic must use P_ prefix: %v", c["command_topic"])
	}
	// Switch without WriteKey keeps the same read/write name.
	sw := b.component("Z1_SOMEWRITE", models.Attributes{EntityType: "switch"}, "x")
	if sw["command_topic"] != "setecna/SYS1/Z1_SOMEWRITE/set" {
		t.Fatalf("non-global command topic should not gain a prefix: %v", sw["command_topic"])
	}
}

func TestOnlyZonesAreDevices(t *testing.T) {
	// Zones map to their own device; everything else stays on the main device.
	if got := elementOf("Z2_TEMP"); got != "Z2" {
		t.Fatalf("Z2_TEMP should map to Z2 device, got %q", got)
	}
	for _, k := range []string{
		"ACS_SET_COMFORT", "GLOBAL_ACS_ENABLE", "GLOBAL_SEASON",
		"C1_RET_TEMP", "S1_TEMP", "HP2_TACS", "HPC_PID_TEMP",
		"EM1_ACCHI", "OT_G0_TEMP", "GLOBAL_OUTPUT_3",
	} {
		if got := elementOf(k); got != "" {
			t.Fatalf("%s should stay on the main device, got %q", k, got)
		}
	}
}

func TestDiagnosticsToggle(t *testing.T) {
	params := models.ParamsMap{
		"Z1_TEMP":    models.Attributes{EntityType: "sensor", DeviceClass: "temperature", Name: "Zone 1 temperature"},
		"HP0_STATUS": models.Attributes{EntityType: "sensor", Name: "Heat pump 0 status"},
	}
	rm := map[string]string{}

	// Diagnostics off (default): the diagnostic entity is a removal (no unique_id),
	// the primary one is untouched.
	off := New("SYS1", nil)
	c := allComponents(t, off, params, rm, false)
	if _, ok := c["HP0_STATUS"]["unique_id"]; ok {
		t.Fatal("diagnostic entity must be removed when diagnostics are off")
	}
	if c["Z1_TEMP"]["unique_id"] == nil {
		t.Fatal("primary entity must stay when diagnostics are off")
	}

	// Diagnostics on: the diagnostic entity is a full (disabled) component.
	on := New("SYS1", nil)
	on.Diagnostics = true
	c = allComponents(t, on, params, rm, false)
	if c["HP0_STATUS"]["unique_id"] == nil {
		t.Fatal("diagnostic entity must be published when diagnostics are on")
	}
	if c["HP0_STATUS"]["enabled_by_default"] != false {
		t.Fatal("diagnostic entity should be disabled by default")
	}
}

func TestMergedSubdeviceCleanup(t *testing.T) {
	b := New("SYS1", nil)
	params := models.ParamsMap{
		"C1_RET_TEMP":     models.Attributes{EntityType: "sensor", Name: "Circuit 1 return temperature"},
		"S1_TEMP":         models.Attributes{EntityType: "sensor", Name: "Source 1 temperature"},
		"HP0_STATUS":      models.Attributes{EntityType: "sensor", Name: "Heat pump 0 status"},
		"HPC_PID_TEMP":    models.Attributes{EntityType: "sensor", Name: "Controller PID temp"},
		"ACS_SET_COMFORT": models.Attributes{EntityType: "number", Name: "ACS comfort setpoint"},
		"Z1_TEMP":         models.Attributes{EntityType: "sensor", DeviceClass: "temperature", Name: "Zone 1 temperature"},
		"GLOBAL_SEASON":   models.Attributes{EntityType: "select", Name: "Season"},
	}
	msgs := b.MergedSubdeviceCleanup(params)

	got := map[string]bool{}
	for _, m := range msgs {
		if len(m.Payload) != 0 {
			t.Fatalf("cleanup payload must be empty (removal), topic %s", m.Topic)
		}
		got[m.Topic] = true
	}
	// Non-zone element sub-devices must be cleaned up.
	for _, elem := range []string{"C1", "S1", "HP0", "HPC", "ACS"} {
		topic := b.configTopicFor("SYS1_" + elem)
		if !got[topic] {
			t.Fatalf("missing cleanup for %s (topic %s)", elem, topic)
		}
	}
	// Zones and plain globals must NOT be cleaned up.
	if got[b.configTopicFor("SYS1_Z1")] {
		t.Fatal("zone Z1 must not be removed")
	}
}

func TestCalendarPrefixRename(t *testing.T) {
	b := New("SYS1", map[string]string{"MT3": "Bagni"})
	// Both the calendar's mode and preset entities follow the MT prefix.
	if got := b.entityLabel("MT3_MODE", models.Attributes{Name: "Calendar 3 mode"}, ""); got != "Bagni mode" {
		t.Fatalf("MT3_MODE label = %q", got)
	}
	if got := b.entityLabel("MT3_FORCING", models.Attributes{Name: "Calendar 3 preset"}, ""); got != "Bagni preset" {
		t.Fatalf("MT3_FORCING label = %q", got)
	}
	// An unrelated calendar is untouched.
	if got := b.entityLabel("MT1_MODE", models.Attributes{Name: "Calendar 1 mode"}, ""); got != "Calendar 1 mode" {
		t.Fatalf("MT1_MODE label = %q", got)
	}
	// An exact-id override still wins over the prefix.
	b2 := New("SYS1", map[string]string{"MT3": "Bagni", "MT3_MODE": "Bagni programma"})
	if got := b2.entityLabel("MT3_MODE", models.Attributes{Name: "Calendar 3 mode"}, ""); got != "Bagni programma" {
		t.Fatalf("exact override should win, got %q", got)
	}
}

func TestSystemControlToggle(t *testing.T) {
	params := models.ParamsMap{
		"GLOBAL_ENABLE":     models.Attributes{EntityType: "switch", Name: "System", EntityCategory: "primary", WriteKey: "P_GLOBAL_ENABLE"},
		"GLOBAL_SEASON":     models.Attributes{EntityType: "select", Name: "Season", EntityCategory: "primary", Options: []string{"winter", "summer"}, WriteKey: "P_GLOBAL_SEASON"},
		"GLOBAL_ACS_ENABLE": models.Attributes{EntityType: "switch", Name: "ACS enable", EntityCategory: "primary", WriteKey: "P_GLOBAL_ACS_ENABLE"},
		"GLOBAL_T_EXT":      models.Attributes{EntityType: "sensor", DeviceClass: "temperature", Name: "Global external temperature"},
	}
	rm := map[string]string{}

	// All controls hidden (bare bridge: all *Control fields false).
	off := New("SYS1", nil)
	off.Diagnostics = true
	c := allComponents(t, off, params, rm, false)
	for _, k := range []string{"GLOBAL_ENABLE", "GLOBAL_SEASON", "GLOBAL_ACS_ENABLE"} {
		if _, ok := c[k]["unique_id"]; ok {
			t.Fatalf("%s must be removed when hidden", k)
		}
	}
	if c["GLOBAL_T_EXT"]["unique_id"] == nil {
		t.Fatal("other entities must stay")
	}

	// All controls shown.
	on := New("SYS1", nil)
	on.Diagnostics = true
	on.SystemControl, on.SeasonControl, on.ACSControl = true, true, true
	c = allComponents(t, on, params, rm, false)
	for _, k := range []string{"GLOBAL_ENABLE", "GLOBAL_SEASON", "GLOBAL_ACS_ENABLE"} {
		if c[k]["unique_id"] == nil {
			t.Fatalf("%s must be present when shown", k)
		}
	}
}

func TestCalendarStateMessages(t *testing.T) {
	// Real values from a live REG system:
	//   giorno -> zones 1,3,4 (CFG1=143 -> clock 1)
	//   notte  -> zones 2,5   (CFG1=159 -> clock 2)
	//   bagni  -> zone 6      (CFG1=173 -> clock 3)
	from := map[string]string{
		"Z1_SENSOR_CHN": "57088", "Z1_CFG1": "143",
		"Z2_SENSOR_CHN": "257", "Z2_CFG1": "159",
		"Z3_SENSOR_CHN": "258", "Z3_CFG1": "143",
		"Z4_SENSOR_CHN": "259", "Z4_CFG1": "143",
		"Z5_SENSOR_CHN": "260", "Z5_CFG1": "159",
		"Z6_SENSOR_CHN": "261", "Z6_CFG1": "173",
		"MT1_XREF": "129", "MT2_XREF": "129", "MT3_XREF": "129",
	}
	b := New("SYS1", map[string]string{"MT1": "giorno", "MT2": "notte", "MT3": "bagni"})
	got := map[string]string{}
	for _, m := range b.CalendarStateMessages(from) {
		got[m.Topic] = m.Payload
	}
	want := map[string]string{
		"setecna/SYS1/Z1_CALENDAR": "giorno",
		"setecna/SYS1/Z2_CALENDAR": "notte",
		"setecna/SYS1/Z3_CALENDAR": "giorno",
		"setecna/SYS1/Z4_CALENDAR": "giorno",
		"setecna/SYS1/Z5_CALENDAR": "notte",
		"setecna/SYS1/Z6_CALENDAR": "bagni",
	}
	for topic, val := range want {
		if got[topic] != val {
			t.Errorf("%s = %q, want %q", topic, got[topic], val)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d messages, want %d: %v", len(got), len(want), got)
	}
}

func TestCalendarFallbackName(t *testing.T) {
	// No MT rename -> "Calendar N"; inactive clock -> em dash.
	from := map[string]string{
		"Z1_SENSOR_CHN": "1", "Z1_CFG1": "143", "MT1_XREF": "129", // clock 1 active
		"Z2_SENSOR_CHN": "1", "Z2_CFG1": "159", "MT2_XREF": "0", // clock 2 inactive
		"Z3_SENSOR_CHN": "1", "Z3_CFG1": "15", // bit7=0 -> not associated
	}
	b := New("SYS1", nil)
	got := map[string]string{}
	for _, m := range b.CalendarStateMessages(from) {
		got[m.Topic] = m.Payload
	}
	if got["setecna/SYS1/Z1_CALENDAR"] != "Calendar 1" {
		t.Errorf("Z1 = %q, want %q", got["setecna/SYS1/Z1_CALENDAR"], "Calendar 1")
	}
	if got["setecna/SYS1/Z2_CALENDAR"] != "—" {
		t.Errorf("Z2 (inactive clock) = %q, want em dash", got["setecna/SYS1/Z2_CALENDAR"])
	}
	if got["setecna/SYS1/Z3_CALENDAR"] != "—" {
		t.Errorf("Z3 (not associated) = %q, want em dash", got["setecna/SYS1/Z3_CALENDAR"])
	}
}

func TestRegimeStateMessages(t *testing.T) {
	// Real night-time capture: all zones automatic (FORCING=0), most in
	// economy (ZONE_SET==SET_ES=245), bagni off (ZONE_SET=0).
	from := map[string]string{
		"Z1_SENSOR_CHN": "1", "Z1_FORCING": "0", "Z1_ZONE_SET": "245",
		"Z1_SET_CS": "240", "Z1_SET_CW": "210", "Z1_SET_ES": "245", "Z1_SET_EW": "190",
		"Z2_SENSOR_CHN": "1", "Z2_FORCING": "0", "Z2_ZONE_SET": "240",
		"Z2_SET_CS": "240", "Z2_SET_CW": "210", "Z2_SET_ES": "245", "Z2_SET_EW": "190",
		"Z3_SENSOR_CHN": "1", "Z3_FORCING": "3", "Z3_ZONE_SET": "245",
		"Z3_SET_CS": "240", "Z3_SET_CW": "210", "Z3_SET_ES": "245", "Z3_SET_EW": "190",
		"Z6_SENSOR_CHN": "1", "Z6_FORCING": "0", "Z6_ZONE_SET": "0",
		"Z6_SET_CS": "240", "Z6_SET_CW": "210", "Z6_SET_ES": "245", "Z6_SET_EW": "190",
	}
	b := New("SYS1", nil) // English
	got := map[string]string{}
	for _, m := range b.RegimeStateMessages(from) {
		got[m.Topic] = m.Payload
	}
	want := map[string]string{
		"setecna/SYS1/Z1_REGIME": "automatic eco",     // auto, ZONE_SET=ES
		"setecna/SYS1/Z2_REGIME": "automatic comfort", // auto, ZONE_SET=CS
		"setecna/SYS1/Z3_REGIME": "forced comfort",    // FORCING!=0, ZONE_SET=ES? no, =245=ES -> eco
		"setecna/SYS1/Z6_REGIME": "off",               // ZONE_SET=0
	}
	// Z3: ZONE_SET=245=ES -> eco, FORCING=3 -> forced -> "forced eco"
	want["setecna/SYS1/Z3_REGIME"] = "forced eco"
	for topic, val := range want {
		if got[topic] != val {
			t.Errorf("%s = %q, want %q", topic, got[topic], val)
		}
	}
}

func TestRegimeLocalizedAndLabels(t *testing.T) {
	b := New("SYS1", nil)
	b.Language = "it"
	from := map[string]string{
		"Z1_SENSOR_CHN": "1", "Z1_FORCING": "0", "Z1_ZONE_SET": "245",
		"Z1_SET_CS": "240", "Z1_SET_ES": "245",
	}
	var payload string
	for _, m := range b.RegimeStateMessages(from) {
		if m.Topic == "setecna/SYS1/Z1_REGIME" {
			payload = m.Payload
		}
	}
	if payload != "automatico eco" {
		t.Errorf("italian regime = %q, want %q", payload, "automatico eco")
	}
	if got := b.localizeLabel("Thermostat"); got != "Termostato" {
		t.Errorf("localizeLabel(Thermostat) it = %q", got)
	}
	if got := b.localizeLabel("Temperature"); got != "Temperatura" {
		t.Errorf("localizeLabel(Temperature) it = %q", got)
	}
}
