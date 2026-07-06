package discovery

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Ingordigia/homeassistant-addon-setecna/models"
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

func TestDeviceConfig(t *testing.T) {
	b := New("SYS1", nil)
	params := make(models.ParamsMap)
	params.AddEnabledParams(testResponseMap(), false)

	msg, err := b.DeviceConfig(params, testResponseMap(), true)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Topic != "homeassistant/device/setecna_SYS1/config" {
		t.Fatalf("wrong topic: %s", msg.Topic)
	}
	if !msg.Retain {
		t.Fatal("discovery payload must be retained")
	}

	var payload struct {
		Device struct {
			Identifiers []string `json:"identifiers"`
		} `json:"device"`
		Origin struct {
			Name string `json:"name"`
		} `json:"origin"`
		AvailabilityTopic string                    `json:"availability_topic"`
		Components        map[string]map[string]any `json:"components"`
	}
	if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if payload.Origin.Name == "" {
		t.Fatal("origin is required by device-based discovery")
	}
	if payload.AvailabilityTopic != "setecna/SYS1/availability" {
		t.Fatalf("wrong availability topic: %s", payload.AvailabilityTopic)
	}

	// Zone 1: climate with humidity, Zone 2: without, Zone 3: absent.
	z1, ok := payload.Components["zone_1"]
	if !ok {
		t.Fatal("zone_1 climate missing")
	}
	if z1["platform"] != "climate" {
		t.Fatalf("zone_1 platform = %v", z1["platform"])
	}
	if z1["unique_id"] != "SYS1_zone_1" {
		t.Fatalf("zone_1 unique_id = %v (must match v1 for seamless migration)", z1["unique_id"])
	}
	if _, ok := z1["current_humidity_topic"]; !ok {
		t.Fatal("zone_1 should expose humidity")
	}
	z2 := payload.Components["zone_2"]
	if _, ok := z2["current_humidity_topic"]; ok {
		t.Fatal("zone_2 should NOT expose humidity")
	}
	if _, ok := payload.Components["zone_3"]; ok {
		t.Fatal("inactive zone_3 must not be discovered")
	}

	// Winter: heat mode and EW/CW setpoints.
	if !strings.Contains(z1["mode_state_template"].(string), "heat") {
		t.Fatal("winter climates must use heat mode")
	}
	if z1["temperature_low_state_topic"] != "setecna/SYS1/Z1_SET_EW" {
		t.Fatalf("winter economy setpoint topic wrong: %v", z1["temperature_low_state_topic"])
	}

	// Every component must have platform + unique_id and state topics
	// outside the discovery prefix.
	for id, cmp := range payload.Components {
		if cmp["platform"] == nil || cmp["unique_id"] == nil {
			t.Fatalf("component %s missing platform/unique_id", id)
		}
		if st, ok := cmp["state_topic"].(string); ok && strings.HasPrefix(st, "homeassistant/") {
			t.Fatalf("component %s keeps state under the discovery prefix: %s", id, st)
		}
		// enum sensors require an options list and must not carry
		// state_class or unit_of_measurement.
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
	msg, err := b.DeviceConfig(models.ParamsMap{}, rm, true)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Components map[string]map[string]any `json:"components"`
	}
	json.Unmarshal([]byte(msg.Payload), &payload)
	z1 := payload.Components["zone_1"]
	if !strings.Contains(z1["action_template"].(string), "cooling") {
		t.Fatal("summer climates must report cooling action")
	}
	if z1["temperature_high_state_topic"] != "setecna/SYS1/Z1_SET_ES" {
		t.Fatalf("summer economy setpoint topic wrong: %v", z1["temperature_high_state_topic"])
	}
}

func TestClimateModeFromForcing(t *testing.T) {
	b := New("SYS1", nil)
	msg, _ := b.DeviceConfig(models.ParamsMap{}, testResponseMap(), true)
	var payload struct {
		Components map[string]map[string]any `json:"components"`
	}
	json.Unmarshal([]byte(msg.Payload), &payload)
	z1 := payload.Components["zone_1"]

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
	msg, _ := b.DeviceConfig(models.ParamsMap{}, testResponseMap(), false)
	var payload struct {
		Components map[string]map[string]any `json:"components"`
	}
	json.Unmarshal([]byte(msg.Payload), &payload)
	up, ok := payload.Components["addon_update"]
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

func TestFriendlyNameOverrides(t *testing.T) {
	names := map[string]string{
		"Z1":              "Bagni",
		"GLOBAL_OUTPUT_3": "Recirculation pump",
	}
	b := New("SYS1", names)

	// Prefix override keeps the role suffix.
	if got := b.friendlyName("Z1_TEMP", "Zone 1 temperature"); got != "Bagni temperature" {
		t.Fatalf("prefix override wrong: %q", got)
	}
	if got := b.friendlyName("Z1_DEWPOINT", "Zone 1 dew point"); got != "Bagni dew point" {
		t.Fatalf("prefix override (dew point) wrong: %q", got)
	}
	// Climate uses the bare name.
	if got := b.friendlyName("Z1", "Zone 1"); got != "Bagni" {
		t.Fatalf("climate override wrong: %q", got)
	}
	// Zones without override are untouched.
	if got := b.friendlyName("Z2_TEMP", "Zone 2 temperature"); got != "Zone 2 temperature" {
		t.Fatalf("non-overridden zone changed: %q", got)
	}
	// Exact-id override.
	if got := b.friendlyName("GLOBAL_OUTPUT_3", "Output 3"); got != "Recirculation pump" {
		t.Fatalf("exact override wrong: %q", got)
	}
	// HPC has no numeric prefix: only exact override applies, else default.
	if got := b.friendlyName("HPC_PID_TEMP", "Heat pump controller PID temperature"); got != "Heat pump controller PID temperature" {
		t.Fatalf("HPC should be untouched by Z1 override: %q", got)
	}

	// The override must appear in the actual discovery payload.
	params := models.ParamsMap{"Z1_TEMP": models.Attributes{EntityType: "sensor", Name: "Zone 1 temperature"}}
	msg, _ := b.DeviceConfig(params, map[string]string{}, false)
	if !strings.Contains(msg.Payload, "Bagni temperature") {
		t.Fatal("override not applied in discovery payload")
	}
}

func TestEntityCategoryRule(t *testing.T) {
	b := New("SYS1", nil)
	// Primary measurements must NOT be diagnostic (stay in main view / usable
	// in the Energy dashboard).
	for _, dc := range []string{"temperature", "humidity", "power", "energy"} {
		c := b.component("X", models.Attributes{EntityType: "sensor", DeviceClass: dc, Name: "x"})
		if _, ok := c["entity_category"]; ok {
			t.Fatalf("%s sensor must not be diagnostic", dc)
		}
	}
	// Raw code sensors (no device class) are diagnostic.
	raw := b.component("HP0_POWER", models.Attributes{EntityType: "sensor", Name: "raw"})
	if raw["entity_category"] != "diagnostic" {
		t.Fatal("raw sensor must be diagnostic")
	}
	// Explicit override always wins.
	ov := b.component("X", models.Attributes{EntityType: "sensor", DeviceClass: "temperature", EntityCategory: "diagnostic", Name: "x"})
	if ov["entity_category"] != "diagnostic" {
		t.Fatal("explicit entity_category must win")
	}
}
