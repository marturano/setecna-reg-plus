package models

import (
	"strings"
	"testing"
)

// snapshot reproduces the relevant subset of a real getres response used to
// validate detection heuristics and sentinel handling.
func snapshot() map[string]string {
	return map[string]string{
		// Heat pumps: 0/2/3 present (real TRIT), 1/4 absent (sentinel TRIT).
		"HP0_TRIT": "0", "HP2_TRIT": "15", "HP3_TRIT": "256",
		"HP1_TRIT": "65280", "HP4_TRIT": "65280",
		// Heat-pump controller present.
		"HPC_NCALD_ACTIVE": "2", "HPC_PID_TEMP": "180", "HPC_REQUIREDPOWER": "500",
		// OpenTherm configured but globally disabled.
		"OT_GLOBAL_ENABLE_R": "0", "OT_GLOBAL_ENABLE_A": "0",
		"OT_G0_ENABLE": "0", "OT_G0_STATUS": "128",
		// Relay outputs.
		"GLOBAL_OUTPUT_0": "255", "GLOBAL_OUTPUT_1": "1", "GLOBAL_OUTPUT_11": "1",
		// Alarms.
		"ANY_ALARM": "0", "ALARM_0": "0", "ALARM_C": "0",
		// One active zone with dew point, one active circuit, one source.
		"Z1_SENSOR_CHN": "57088", "Z1_DEWPOINT": "128",
		"C1_TEMP": "258", "C1_OUTPUT_PA": "1", "C1_RET_TEMP": "32769",
		"S1_DESCR": "197", "S1_TEMP": "258", "S1_STATUS": "207",
	}
}

func TestHeatPumpDetection(t *testing.T) {
	m := make(ParamsMap)
	m.addHeatPumps(snapshot(), true, true, false)
	for _, i := range []string{"0", "2", "3"} {
		if _, ok := m["HP"+i+"_TRIT"]; !ok {
			t.Fatalf("heat pump %s should be present", i)
		}
	}
	for _, i := range []string{"1", "4"} {
		if _, ok := m["HP"+i+"_TRIT"]; ok {
			t.Fatalf("heat pump %s should be absent (sentinel)", i)
		}
	}
	// TRIT must be a scaled temperature with sentinel filtering.
	if a := m["HP2_TRIT"]; a.DeviceClass != "temperature" || !strings.Contains(a.ValueTemplate, "65535") {
		t.Fatalf("HP2_TRIT not a sentinel-filtered temperature: %+v", a)
	}
	// POWER is raw (unit unknown) - must not claim a device class.
	if a := m["HP2_POWER"]; a.DeviceClass != "" || a.UnitOfMeasurement != "" {
		t.Fatalf("HP2_POWER should be a raw sensor, got %+v", a)
	}
}

func TestHeatPumpControllerGating(t *testing.T) {
	m := make(ParamsMap)
	m.addHeatPumpController(snapshot(), true, true, false)
	if _, ok := m["HPC_PID_TEMP"]; !ok {
		t.Fatal("HPC_PID_TEMP should be present when controller exists")
	}
	if _, ok := m["HPC_NCALD_ACTIVE"]; !ok {
		t.Fatal("HPC_NCALD_ACTIVE should be present")
	}
	// Absent controller -> nothing created.
	empty := make(ParamsMap)
	empty.addHeatPumpController(map[string]string{}, true, true, false)
	if len(empty) != 0 {
		t.Fatalf("no HPC entities expected without controller, got %d", len(empty))
	}
}

func TestOpenThermGatedOff(t *testing.T) {
	m := make(ParamsMap)
	m.addOpenTherm(snapshot(), true, true, false)
	if len(m) != 0 {
		t.Fatalf("OpenTherm disabled -> no entities, got %d", len(m))
	}
	// Enabled system with one active generator.
	on := map[string]string{"OT_GLOBAL_ENABLE_R": "1", "OT_G0_ENABLE": "1"}
	m2 := make(ParamsMap)
	m2.addOpenTherm(on, true, true, false)
	if _, ok := m2["OT_G0_TMAND"]; !ok {
		t.Fatal("enabled generator 0 should expose flow temperature")
	}
}

func TestGlobalOutputsAndAlarms(t *testing.T) {
	m := make(ParamsMap)
	m.addGlobalOutputs(snapshot(), true, true, false)
	if a := m["GLOBAL_OUTPUT_1"]; a.EntityType != "binary_sensor" {
		t.Fatalf("output should be binary_sensor, got %+v", a)
	}
	m.addSystemAlarms(snapshot(), true, true, false)
	if a := m["ANY_ALARM"]; a.DeviceClass != "problem" {
		t.Fatalf("ANY_ALARM should be a problem binary_sensor, got %+v", a)
	}
	if _, ok := m["ALARM_C"]; !ok {
		t.Fatal("hex-suffixed alarm word ALARM_C should be handled")
	}
}

func TestZoneDewpointAndCircuitSource(t *testing.T) {
	m := make(ParamsMap)
	m.AddEnabledParams(snapshot(), true)
	if a := m["Z1_DEWPOINT"]; a.DeviceClass != "temperature" || !strings.Contains(a.ValueTemplate, "32769") {
		t.Fatalf("Z1_DEWPOINT not a sentinel-filtered temperature: %+v", a)
	}
	if a := m["C1_OUTPUT_PA"]; a.EntityType != "binary_sensor" {
		t.Fatalf("circuit pump A should be a binary_sensor, got %+v", a)
	}
	if _, ok := m["C1_RET_TEMP"]; !ok {
		t.Fatal("circuit return temperature missing")
	}
	if _, ok := m["S1_TEMP"]; !ok {
		t.Fatal("source temperature missing")
	}
}

func TestEnergyMeterHighWord(t *testing.T) {
	m := make(ParamsMap)
	m.addEnergymeters(map[string]string{}, true, true, false)
	if _, ok := m["EM1_ACCHI"]; !ok {
		t.Fatal("EM1_ACCHI (high word) missing")
	}
	if _, ok := m["EM4_ACC2HI"]; !ok {
		t.Fatal("EM4_ACC2HI (export high word) missing")
	}
}
