package models

import (
	"fmt"
	"regexp"
)

type Attributes struct {
	CommandTemplate   string   `json:"command_template"`
	DeviceClass       string   `json:"device_class"`
	EntityCategory    string   `json:"entity_category"`
	EntityType        string   `json:"entity_type"`
	Max               float64  `json:"max"`
	Min               float64  `json:"min"`
	Name              string   `json:"name"`
	Options           []string `json:"options"`
	StateClass        string   `json:"state_class"`
	Step              float64  `json:"step"`
	UnitOfMeasurement string   `json:"unit_of_measurement"`
	ValueTemplate     string   `json:"value_template"`
	// WriteKey overrides the parameter name used when writing back to Setecna.
	// The Setecna cloud reads global params as "GLOBAL_*" but only accepts
	// writes to their "P_GLOBAL_*" counterpart; the state topic keeps the
	// read name so unique_id and history are unaffected.
	WriteKey string `json:"write_key"`
	// StateKey overrides the parameter whose topic the entity subscribes to.
	// Used for derived sensors that read another parameter's value (e.g. the
	// zone regime sensor reads the FORCING topic). The entity keeps its own
	// unique_id from its map key.
	StateKey string `json:"state_key"`
}

// Sentinel values used by the REG controller to signal "not available"
// on 8-bit (255), 16-bit signed (32768/32769) and 16-bit unsigned
// (65280/65535) channels. Templates below blank the state on these values
// so the entity stays "unknown" instead of showing a garbage number.
const (
	// tplTemp16Sentinel: scaled temperature (value/10 °C) for 16-bit probe
	// channels. Filters only the 16-bit "not available" sentinels; 255 is a
	// valid reading here (25.5 °C), so it must NOT be blanked.
	tplTemp16Sentinel = `{% set v = value | int %}{% if v not in [32768, 32769, 65280, 65535] %}{{ v / 10 }}{% endif %}`
	// tplTempSigned: scaled temperature (value/10 °C) for channels that can go
	// negative (external temperature, dewpoint). The value is a signed 16-bit
	// integer, so values >= 32768 are negative (v - 65536). Only 32768/32769
	// (-3276.8/-3276.7 °C, impossible) are treated as "not available"; the high
	// unsigned range is valid negative temperatures (e.g. 65535 = -0.1 °C).
	tplTempSigned = `{% set v = value | int %}{% if v not in [32768, 32769] %}{{ (v - 65536 if v >= 32768 else v) / 10 }}{% endif %}`
	// tplRawSentinel: raw integer with sentinel filter (unit/scale unknown).
	tplRawSentinel = `{% set v = value | int %}{% if v not in [255, 32768, 32769, 65280, 65535] %}{{ v }}{% endif %}`
	// tplOnOff: boolean from "1"/other.
	tplOnOff = `{% if value == "1" %}on{% else %}off{% endif %}`
)

type ParamsMap map[string]Attributes

func (m ParamsMap) AddEnabledParams(from map[string]string, isReadOnly bool) {
	m.addLastUpdate(from, true, isReadOnly, !isReadOnly)
	m.addGlobals(from, true, isReadOnly, !isReadOnly)
	m.addDomesticHotWater(from, true, isReadOnly, !isReadOnly)
	m.addAnalogInput(from, true, isReadOnly, !isReadOnly)
	m.addDigitalInput(from, true, isReadOnly, !isReadOnly)
	m.addDigitalAlarm(from, true, isReadOnly, !isReadOnly)
	m.addZones(from, true, isReadOnly, !isReadOnly)
	m.addCircuits(from, true, isReadOnly, !isReadOnly)
	m.addSources(from, true, isReadOnly, !isReadOnly)
	m.addDehumidifier(from, true, isReadOnly, !isReadOnly)
	m.addEnergymeters(from, true, isReadOnly, !isReadOnly)
	m.addCalendars(from, true, isReadOnly, !isReadOnly)
	m.addHeatPumps(from, true, isReadOnly, !isReadOnly)
	m.addHeatPumpController(from, true, isReadOnly, !isReadOnly)
	m.addOpenTherm(from, true, isReadOnly, !isReadOnly)
	m.addGlobalOutputs(from, true, isReadOnly, !isReadOnly)
	m.addSystemAlarms(from, true, isReadOnly, !isReadOnly)
}

func (m ParamsMap) AddDisabledParams(from map[string]string, isReadOnly bool) {
	m.addGlobals(from, false, !isReadOnly, isReadOnly)
	m.addDomesticHotWater(from, false, !isReadOnly, isReadOnly)
	m.addAnalogInput(from, false, !isReadOnly, isReadOnly)
	m.addDigitalInput(from, false, !isReadOnly, isReadOnly)
	m.addDigitalAlarm(from, false, !isReadOnly, isReadOnly)
	m.addZones(from, false, !isReadOnly, isReadOnly)
	m.addCircuits(from, false, !isReadOnly, isReadOnly)
	m.addSources(from, false, !isReadOnly, isReadOnly)
	m.addDehumidifier(from, false, !isReadOnly, isReadOnly)
	m.addEnergymeters(from, false, !isReadOnly, isReadOnly)
	m.addCalendars(from, false, !isReadOnly, isReadOnly)
	m.addHeatPumps(from, false, !isReadOnly, isReadOnly)
	m.addHeatPumpController(from, false, !isReadOnly, isReadOnly)
	m.addOpenTherm(from, false, !isReadOnly, isReadOnly)
	m.addGlobalOutputs(from, false, !isReadOnly, isReadOnly)
	m.addSystemAlarms(from, false, !isReadOnly, isReadOnly)
}

func (m ParamsMap) addLastUpdate(from map[string]string, static, read, write bool) {
	if static {
		m["LAST_UPDATE"] = Attributes{
			DeviceClass: "timestamp",
			EntityType:  "sensor",
			Name:        "Last update",
		}
	}
}

func (m ParamsMap) addGlobals(from map[string]string, static, read, write bool) {
	if static {
		if write {
			// Writable master on/off for the whole plant. The cloud accepts
			// writes only on P_GLOBAL_ENABLE (0 = off, 1 = on); state is read
			// from GLOBAL_ENABLE.
			m["GLOBAL_ENABLE"] = Attributes{
				Name:           "System",
				EntityType:     "switch",
				EntityCategory: "primary",
				WriteKey:       "P_GLOBAL_ENABLE",
			}
		} else {
			m["GLOBAL_ENABLE"] = Attributes{
				Name:          "Global state",
				EntityType:    "binary_sensor",
				ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
			}
		}
		// The controller's own external-temperature input is often not wired;
		// in that case it reads a "not available" sentinel. Only expose it when
		// it carries a real reading, otherwise the outside temperature comes
		// from the heat pump probe (HP<n>_TEXT).
		switch from["GLOBAL_T_EXT"] {
		case "", "255", "32768", "32769", "65280", "65535":
			// no valid controller external probe: skip
		default:
			m["GLOBAL_T_EXT"] = Attributes{
				Name:              "Global external temperature",
				EntityType:        "sensor",
				DeviceClass:       "temperature",
				UnitOfMeasurement: "°C",
				StateClass:        "measurement",
				ValueTemplate:     tplTempSigned,
			}
		}
		if write {
			// Writable season selector. The cloud accepts writes only on
			// P_GLOBAL_SEASON (0 = winter, 1 = summer); state is read from
			// GLOBAL_SEASON.
			m["GLOBAL_SEASON"] = Attributes{
				Name:            "Season",
				EntityType:      "select",
				EntityCategory:  "primary",
				Options:         []string{"winter", "summer"},
				ValueTemplate:   "{% if value == \"1\" %}summer{% else %}winter{% endif %}",
				CommandTemplate: "{% if value == \"summer\" %}1{% else %}0{% endif %}",
				WriteKey:        "P_GLOBAL_SEASON",
			}
		} else {
			m["GLOBAL_SEASON"] = Attributes{
				Name:          "Global season",
				EntityType:    "sensor",
				DeviceClass:   "enum",
				ValueTemplate: "{% if value == \"0\" %}winter{% elif value == \"1\" %}summer{% else %}{{ value }}{% endif %}",
				Options:       []string{"winter", "summer"},
			}
		}
		m["GLOBAL_DEICING"] = Attributes{
			Name:          "Global de-ice state",
			EntityType:    "binary_sensor",
			ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
		}
		m["GLOBAL_EXPECTED_DEWP"] = Attributes{
			Name:              "Global dewpoint",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTempSigned,
		}
	}
	if read {
		m["GLOBAL_ZONE_T_HYST"] = Attributes{
			Name:              "Global zone temperature hysteresis",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
		m["GLOBAL_ZONE_RH_HYST"] = Attributes{
			Name:              "Global zone humidity hysteresis",
			EntityType:        "sensor",
			DeviceClass:       "humidity",
			UnitOfMeasurement: "%",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
		m["GLOBAL_ZONE_DEICE_TRESH"] = Attributes{
			Name:              "Global zone de-ice threshold",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
	}
	if write {
		m["GLOBAL_ZONE_T_HYST"] = Attributes{
			Name:              "Global zone temperature hysteresis",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               1,
			Min:               0.1,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
		m["GLOBAL_ZONE_RH_HYST"] = Attributes{
			Name:              "Global zone humidity hysteresis",
			EntityType:        "number",
			DeviceClass:       "humidity",
			UnitOfMeasurement: "%",
			Max:               5,
			Min:               1,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
		m["GLOBAL_ZONE_DEICE_TRESH"] = Attributes{
			Name:              "Global zone de-ice threshold",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               10,
			Min:               6,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
	}
}

func (m ParamsMap) addDomesticHotWater(from map[string]string, static, read, write bool) {
	if static {
		m["ACS_MAIN_OUTPUT"] = Attributes{
			Name:          "ACS state",
			EntityType:    "binary_sensor",
			ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
		}
		if write {
			// Master on/off for domestic hot water. Global param: the cloud
			// accepts writes only on P_GLOBAL_ACS_ENABLE (0 = off, 1 = on).
			m["GLOBAL_ACS_ENABLE"] = Attributes{
				Name:           "ACS enable",
				EntityType:     "switch",
				EntityCategory: "primary",
				WriteKey:       "P_GLOBAL_ACS_ENABLE",
			}
		} else {
			m["GLOBAL_ACS_ENABLE"] = Attributes{
				Name:          "ACS enabled",
				EntityType:    "binary_sensor",
				ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
			}
		}
		m["GLOBAL_T_ACS"] = Attributes{
			Name:              "ACS temperature",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
		m["GLOBAL_SET_ACS"] = Attributes{
			Name:              "ACS active setpoint",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
	}
	if read {
		m["ACS_SET_ECONOMY"] = Attributes{
			Name:              "ACS economy setpoint",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
		m["ACS_SET_COMFORT"] = Attributes{
			Name:              "ACS comfort setpoint",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
		m["ACS_SET_HYST"] = Attributes{
			Name:              "ACS setpoint hysteresis",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
		m["ACS_SET_DELTA"] = Attributes{
			Name:              "ACS second stage deviation",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
	}
	if write {
		m["ACS_SET_ECONOMY"] = Attributes{
			Name:              "ACS economy setpoint",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               60,
			Min:               30,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
		m["ACS_SET_COMFORT"] = Attributes{
			Name:              "ACS comfort setpoint",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               60,
			Min:               30,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
		m["ACS_SET_HYST"] = Attributes{
			Name:              "ACS setpoint hysteresis",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               10,
			Min:               0,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
		m["ACS_SET_DELTA"] = Attributes{
			Name:              "ACS second stage deviation",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               10,
			Min:               0,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
	}
}

func (m ParamsMap) addAnalogInput(from map[string]string, static, read, write bool) {
	if static {
		for i := 1; i <= 8; i++ {
			if from["FAIN"+fmt.Sprint(i)+"_TEMP"] != "32769" {
				m["FAIN"+fmt.Sprint(i)+"_TEMP"] = Attributes{
					Name:              "Analog input " + fmt.Sprint(i),
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					StateClass:        "measurement",
					UnitOfMeasurement: "°C",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "",
				}
			}
		}
	}
}

func (m ParamsMap) addDigitalInput(from map[string]string, static, read, write bool) {
	if static {
		for i := 1; i <= 8; i++ {
			m["FDIN"+fmt.Sprint(i)+"_STATUS"] = Attributes{
				Name:          "Digital input " + fmt.Sprint(i),
				EntityType:    "binary_sensor",
				ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
			}
		}
	}
}

func (m ParamsMap) addDigitalAlarm(from map[string]string, static, read, write bool) {
	if static {
		for i := 1; i <= 5; i++ {
			m["FALDIN"+fmt.Sprint(i)+"_STATUS"] = Attributes{
				Name:          "Alarm " + fmt.Sprint(i),
				EntityType:    "binary_sensor",
				ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
			}
		}
	}
}

func (m ParamsMap) addZones(from map[string]string, static, read, write bool) {
	for i := 1; i <= 32; i++ {
		if from["Z"+fmt.Sprint(i)+"_SENSOR_CHN"] != "0" {
			if static {
				m["Z"+fmt.Sprint(i)+"_OUTPUT"] = Attributes{
					Name:          "Zone " + fmt.Sprint(i) + " state",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}
				m["Z"+fmt.Sprint(i)+"_TEMP"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " temperature",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_DEWPOINT"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " dew point",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     tplTemp16Sentinel,
				}
				// Current regime, derived from FORCING (which encodes both the
				// forcing and the schedule-driven regime on this controller):
				// 1=off, 2=forced economy, 3=forced comfort, 0=automatic (idle),
				// automatic active = 48 + regime (48/49=off, 50=economy, 51=comfort).
				m["Z"+fmt.Sprint(i)+"_REGIME"] = Attributes{
					Name:           "Zone " + fmt.Sprint(i) + " regime",
					EntityType:     "sensor",
					DeviceClass:    "enum",
					EntityCategory: "primary",
					StateKey:       "Z" + fmt.Sprint(i) + "_FORCING",
					ValueTemplate:  "{% if value == \"2\" %}forced economy{% elif value == \"3\" %}forced comfort{% elif value == \"50\" %}automatic economy{% elif value == \"51\" %}automatic comfort{% elif value in [\"1\", \"48\", \"49\"] %}off{% else %}automatic{% endif %}",
					Options:        []string{"automatic", "automatic economy", "automatic comfort", "forced economy", "forced comfort", "off"},
				}
				// Associated clock (Orologio / calendar). The clock index is
				// encoded in bits 4-6 of Z<n>_CFG1 (bit 7 = "clock associated").
				// The human name (giorno/notte/bagni) is resolved and published
				// by the bridge (see CalendarStateMessages), so this is a plain
				// text sensor reading its own dedicated topic.
				m["Z"+fmt.Sprint(i)+"_CALENDAR"] = Attributes{
					Name:           "Zone " + fmt.Sprint(i) + " calendar",
					EntityType:     "sensor",
					EntityCategory: "primary",
				}
				// Raw ZONE_MODE kept as a hidden diagnostic (its encoding cannot
				// tell automatic economy from comfort - both read 0 - so FORCING
				// is used for the regime above).
				m["Z"+fmt.Sprint(i)+"_ZONE_MODE"] = Attributes{
					Name:          "Zone " + fmt.Sprint(i) + " mode (raw)",
					EntityType:    "sensor",
					ValueTemplate: tplRawSentinel,
				}
				m["Z"+fmt.Sprint(i)+"_ZONE_SET"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
			}
			if read {
				m["Z"+fmt.Sprint(i)+"_FORCING"] = Attributes{
					Name:          "Zone " + fmt.Sprint(i) + " forcing",
					EntityType:    "sensor",
					DeviceClass:   "enum",
					ValueTemplate: "{% if value == \"0\" %}automatic{% elif value == \"1\" %}off{% elif value == \"2\" %}economy{% elif value == \"3\" %}comfort{% else %}{{ value }}{% endif %}",
					Options:       []string{"automatic", "off", "economy", "comfort"},
				}
				m["Z"+fmt.Sprint(i)+"_SET_CW"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " C.W. setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_EW"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " E.W. setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_CS"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " C.S. setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_ES"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " E.S. setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
			}
			if write {
				m["Z"+fmt.Sprint(i)+"_FORCING"] = Attributes{
					Name:            "Zone " + fmt.Sprint(i) + " forcing",
					EntityType:      "select",
					Options:         []string{"automatic", "off", "economy", "comfort"},
					ValueTemplate:   "{% if value == \"1\" %}off{% elif value == \"2\" %}economy{% elif value == \"3\" %}comfort{% else %}automatic{% endif %}",
					CommandTemplate: "{% if value == \"off\" %}1{% elif value == \"economy\" %}2{% elif value == \"comfort\" %}3{% else %}0{% endif %}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_CW"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " C.W. setpoint",
					EntityType:        "number",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					Max:               30,
					Min:               15,
					Step:              0.1,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_EW"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " E.W. setpoint",
					EntityType:        "number",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					Max:               30,
					Min:               15,
					Step:              0.1,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_CS"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " C.S. setpoint",
					EntityType:        "number",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					Max:               30,
					Min:               15,
					Step:              0.1,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_ES"] = Attributes{
					Name:              "Zone " + fmt.Sprint(i) + " E.S. setpoint",
					EntityType:        "number",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					Max:               30,
					Min:               15,
					Step:              0.1,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
			}
			if from["Z"+fmt.Sprint(i)+"_RH"] != "32769" {
				if static {
					m["Z"+fmt.Sprint(i)+"_RH"] = Attributes{
						Name:              "Zone " + fmt.Sprint(i) + " humidity",
						EntityType:        "sensor",
						DeviceClass:       "humidity",
						UnitOfMeasurement: "%",
						StateClass:        "measurement",
						ValueTemplate:     "{{ value | int / 10 }}",
						CommandTemplate:   "{{ (value * 10) | int }}",
					}
				}
				if read {
					m["Z"+fmt.Sprint(i)+"_SET_RH"] = Attributes{
						Name:              "Zone " + fmt.Sprint(i) + " humidity setpoint",
						EntityType:        "sensor",
						DeviceClass:       "humidity",
						UnitOfMeasurement: "%",
						StateClass:        "measurement",
						ValueTemplate:     "{{ value | int / 10 }}",
						CommandTemplate:   "{{ (value * 10) | int }}",
					}
				}
				if write {
					m["Z"+fmt.Sprint(i)+"_SET_RH"] = Attributes{
						Name:              "Zone " + fmt.Sprint(i) + " humidity setpoint",
						EntityType:        "number",
						DeviceClass:       "humidity",
						UnitOfMeasurement: "%",
						Max:               70,
						Min:               40,
						Step:              0.1,
						StateClass:        "measurement",
						ValueTemplate:     "{{ value | int / 10 }}",
						CommandTemplate:   "{{ (value * 10) | int }}",
					}
				}
			}
		}
	}
}

func (m ParamsMap) addCircuits(from map[string]string, static, read, write bool) {
	for i := 1; i <= 8; i++ {
		if from["C"+fmt.Sprint(i)+"_TEMP"] != "32769" {
			if static {
				m["C"+fmt.Sprint(i)+"_TEMP"] = Attributes{
					Name:              "Circuit " + fmt.Sprint(i) + " temperature",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["C"+fmt.Sprint(i)+"_SET"] = Attributes{
					Name:              "Circuit " + fmt.Sprint(i) + " temperature setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["C"+fmt.Sprint(i)+"_RET_TEMP"] = Attributes{
					Name:              "Circuit " + fmt.Sprint(i) + " return temperature",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     tplTemp16Sentinel,
				}
				m["C"+fmt.Sprint(i)+"_OUTPUT_PA"] = Attributes{
					Name:          "Circuit " + fmt.Sprint(i) + " pump A",
					EntityType:    "binary_sensor",
					ValueTemplate: tplOnOff,
				}
				m["C"+fmt.Sprint(i)+"_OUTPUT_PB"] = Attributes{
					Name:          "Circuit " + fmt.Sprint(i) + " pump B",
					EntityType:    "binary_sensor",
					ValueTemplate: tplOnOff,
				}
			}
		}
	}
}

func (m ParamsMap) addSources(from map[string]string, static, read, write bool) {
	for i := 1; i <= 3; i++ {
		if from["S"+fmt.Sprint(i)+"_DESCR"] != "0" {
			if static {
				m["S"+fmt.Sprint(i)+"_ENABLED"] = Attributes{
					Name:          "Source " + fmt.Sprint(i) + " enabled",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}
				m["S"+fmt.Sprint(i)+"_OUTPUT"] = Attributes{
					Name:          "Source " + fmt.Sprint(i) + " state",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}
				m["S"+fmt.Sprint(i)+"_AUXOUTPUT"] = Attributes{
					Name:          "Source " + fmt.Sprint(i) + " auxiliary state",
					EntityType:    "binary_sensor",
					ValueTemplate: tplOnOff,
				}
				m["S"+fmt.Sprint(i)+"_TEMP"] = Attributes{
					Name:              "Source " + fmt.Sprint(i) + " temperature",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     tplTemp16Sentinel,
				}
				m["S"+fmt.Sprint(i)+"_AUXTEMP"] = Attributes{
					Name:              "Source " + fmt.Sprint(i) + " auxiliary temperature",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     tplTemp16Sentinel,
				}
				m["S"+fmt.Sprint(i)+"_SET"] = Attributes{
					Name:              "Source " + fmt.Sprint(i) + " setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     tplTemp16Sentinel,
				}
				// Raw status code (meaning not reverse engineered yet).
				m["S"+fmt.Sprint(i)+"_STATUS"] = Attributes{
					Name:          "Source " + fmt.Sprint(i) + " status code",
					EntityType:    "sensor",
					ValueTemplate: tplRawSentinel,
				}
			}
		}
	}
}

func (m ParamsMap) addDehumidifier(from map[string]string, static, read, write bool) {
	for i := 1; i <= 8; i++ {
		if from["D"+fmt.Sprint(i)+"_SPEED_LOW"] != "0" && from["D"+fmt.Sprint(i)+"_SPEED_ECONOMY"] != "0" {
			if static {
				m["D"+fmt.Sprint(i)+"_OUTPUT_RENEW"] = Attributes{
					Name:          "Fan " + fmt.Sprint(i) + " renew",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}
				m["D"+fmt.Sprint(i)+"_OUTPUT_DEUM"] = Attributes{
					Name:          "Fan " + fmt.Sprint(i) + " dehumidify",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}

			}
			if read {
				m["D"+fmt.Sprint(i)+"_SPEED_LOW"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " low flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_MED"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " medium flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_HIGH"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " high flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_BOOST"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " boost flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_ECONOMY"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " economy flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_COMFORT"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " comfort flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
			}
			if write {
				m["D"+fmt.Sprint(i)+"_SPEED_LOW"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " low flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_MED"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " medium flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_HIGH"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " high flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_BOOST"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " boost flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_ECONOMY"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " economy flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_COMFORT"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " comfort flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
			}
		}
	}
}

func (m ParamsMap) addEnergymeters(from map[string]string, static, read, write bool) {
	for i := 1; i <= 4; i++ {
		if static {
			m["EM"+fmt.Sprint(i)+"_INSTANT"] = Attributes{
				Name:              "Energy meter " + fmt.Sprint(i) + " power",
				EntityType:        "sensor",
				DeviceClass:       "power",
				UnitOfMeasurement: "kW",
				StateClass:        "measurement",
				ValueTemplate:     "{% set v = value | int %}{% if v in [255, 32768, 32769, 65280, 65535] %}{% elif v >= 32768 %}{{ (v - 65536) / 100 }}{% else %}{{ v / 100 }}{% endif %}",
				CommandTemplate:   "",
			}
			m["EM"+fmt.Sprint(i)+"_ACCLO"] = Attributes{
				Name:              "Energy meter " + fmt.Sprint(i) + " total energy import",
				EntityType:        "sensor",
				DeviceClass:       "energy",
				UnitOfMeasurement: "kWh",
				StateClass:        "total_increasing",
				ValueTemplate:     "{% set v = value | int %}{% if v not in [255, 32768, 32769, 65280, 65535] %}{{ v / 10 }}{% endif %}",
				CommandTemplate:   "",
			}
			// High word of the 32-bit import accumulator. Exposed as a raw
			// diagnostic value so a template sensor can reconstruct the full
			// total as (ACCHI * 65536 + ACCLO) / 10 (see DOCS.md).
			m["EM"+fmt.Sprint(i)+"_ACCHI"] = Attributes{
				Name:          "Energy meter " + fmt.Sprint(i) + " total energy import (high word)",
				EntityType:    "sensor",
				ValueTemplate: tplRawSentinel,
			}
			if i == 4 {
				m["EM"+fmt.Sprint(i)+"_ACC2LO"] = Attributes{
					Name:              "Energy meter " + fmt.Sprint(i) + " total energy export",
					EntityType:        "sensor",
					DeviceClass:       "energy",
					UnitOfMeasurement: "kWh",
					StateClass:        "total_increasing",
					ValueTemplate:     "{% set v = value | int %}{% if v not in [255, 32768, 32769, 65280, 65535] %}{{ v / 10 }}{% endif %}",
					CommandTemplate:   "",
				}
				m["EM"+fmt.Sprint(i)+"_ACC2HI"] = Attributes{
					Name:          "Energy meter " + fmt.Sprint(i) + " total energy export (high word)",
					EntityType:    "sensor",
					ValueTemplate: tplRawSentinel,
				}
			}
		}

	}
}

func (m ParamsMap) addCalendars(from map[string]string, static, read, write bool) {
	for i := 1; i <= 8; i++ {
		if from["MT"+fmt.Sprint(i)+"_XREF"] != "0" {
			if static {
				m["MT"+fmt.Sprint(i)+"_MODE"] = Attributes{
					Name:          "Calendar " + fmt.Sprint(i) + " mode",
					EntityType:    "sensor",
					DeviceClass:   "enum",
					ValueTemplate: "{% if value == \"1\" %}off{% elif value == \"2\" %}economy{% elif value == \"3\" %}comfort{% else %}{{ value }}{% endif %}",
					Options:       []string{"off", "economy", "comfort"},
				}
			}
			if read {
				m["MT"+fmt.Sprint(i)+"_FORCING"] = Attributes{
					Name:          "Calendar " + fmt.Sprint(i) + " preset",
					EntityType:    "sensor",
					DeviceClass:   "enum",
					ValueTemplate: "{% if value == \"0\" %}automatic{% elif value == \"1\" %}forced off{% elif value == \"2\" %}forced economy{% elif value == \"3\" %}forced comfort{% else %}{{ value }}{% endif %}",
					Options:       []string{"automatic", "forced off", "forced economy", "forced comfort"},
				}
			}
			if write {
				m["MT"+fmt.Sprint(i)+"_FORCING"] = Attributes{
					Name:            "Calendar " + fmt.Sprint(i) + " preset",
					Options:         []string{"automatic", "forced off", "forced economy", "forced comfort"},
					EntityType:      "select",
					ValueTemplate:   "{% if value == \"1\" %}forced off{% elif value == \"2\" %}forced economy{% elif value == \"3\" %}forced comfort{% else %}automatic{% endif %}",
					CommandTemplate: "{% if value == \"forced off\" %}1{% elif value == \"forced economy\" %}2{% elif value == \"forced comfort\" %}3{% else %}0{% endif %}",
				}

			}
		}
	}
}

// addHeatPumps exposes the heat-pump units (HP0..HP4) as read-only
// diagnostic sensors. A unit is considered present when its return
// temperature is not a "not available" sentinel. Temperatures are scaled
// by 10; fields whose unit could not be reverse engineered are exposed as
// raw values and clearly labelled as such.
func (m ParamsMap) addHeatPumps(from map[string]string, static, read, write bool) {
	if !static {
		return
	}
	temp := func(id, label string, i int) {
		m[id] = Attributes{
			Name:              "Heat pump " + fmt.Sprint(i) + " " + label,
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
	}
	raw := func(id, label string, i int) {
		m[id] = Attributes{
			Name:          "Heat pump " + fmt.Sprint(i) + " " + label,
			EntityType:    "sensor",
			ValueTemplate: tplRawSentinel,
		}
	}
	// tempSigned is like temp but for channels that can go negative (the
	// outside-temperature probe).
	tempSigned := func(id, label string, i int) {
		m[id] = Attributes{
			Name:              "Heat pump " + fmt.Sprint(i) + " " + label,
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTempSigned,
		}
	}
	for i := 0; i <= 4; i++ {
		p := "HP" + fmt.Sprint(i)
		trit := from[p+"_TRIT"]
		// Present only when the return temperature is a real reading (not a
		// "not available" sentinel).
		switch trit {
		case "", "255", "32768", "32769", "65280", "65535":
			continue
		}
		temp(p+"_TRIT", "return temperature", i)
		tempSigned(p+"_TEXT", "outside temperature", i)
		temp(p+"_TMAND", "flow temperature", i)
		temp(p+"_TACS", "ACS temperature", i)
		// Raw fields: exact unit/encoding not yet reverse engineered.
		raw(p+"_STATUS", "status code", i)
		raw(p+"_POWER", "power (raw)", i)
		raw(p+"_RQ", "request (raw)", i)
		raw(p+"_OEMERROR", "OEM error code", i)
		raw(p+"_OEMSTATUS", "OEM status code", i)
	}
}

// addHeatPumpController exposes the heat-pump cascade controller (HPC_*).
// Only created when the controller is present (its NCALD_ACTIVE key exists).
// PID_TEMP is a temperature; the remaining fields are raw because their
// units are not documented.
func (m ParamsMap) addHeatPumpController(from map[string]string, static, read, write bool) {
	if !static || from["HPC_NCALD_ACTIVE"] == "" {
		return
	}
	m["HPC_PID_TEMP"] = Attributes{
		Name:              "Heat pump controller PID temperature",
		EntityType:        "sensor",
		DeviceClass:       "temperature",
		UnitOfMeasurement: "°C",
		StateClass:        "measurement",
		ValueTemplate:     tplTemp16Sentinel,
	}
	measurement := func(id, label string) {
		m[id] = Attributes{
			Name:          "Heat pump controller " + label,
			EntityType:    "sensor",
			StateClass:    "measurement",
			ValueTemplate: tplRawSentinel,
		}
	}
	raw := func(id, label string) {
		m[id] = Attributes{
			Name:          "Heat pump controller " + label,
			EntityType:    "sensor",
			ValueTemplate: tplRawSentinel,
		}
	}
	measurement("HPC_NCALD_ACTIVE", "active stages")
	// Required power: raw value is in hundredths of a kW (500 -> 5.00 kW),
	// confirmed against 5 kW nominal heat-pump stages on a real system.
	m["HPC_REQUIREDPOWER"] = Attributes{
		Name:              "Heat pump controller required power",
		EntityType:        "sensor",
		DeviceClass:       "power",
		UnitOfMeasurement: "kW",
		StateClass:        "measurement",
		ValueTemplate:     `{% set v = value | int %}{% if v not in [255, 32768, 32769, 65280, 65535] %}{{ v / 100 }}{% endif %}`,
	}
	measurement("HPC_PID_OUTPUT", "PID output (raw)")
	measurement("HPC_GRACETIMER", "grace timer (raw)")
	raw("HPC_REQUEST_R", "heating request (raw)")
	raw("HPC_REQUEST_ACS", "ACS request (raw)")
	raw("HPC_FLAGS", "flags (raw)")
	raw("HPC_PID_THERMOSTAT", "PID thermostat (raw)")
}

// addOpenTherm exposes the OpenTherm generator cascade (OT_G0..OT_G8).
// The whole block is gated on the OpenTherm subsystem being enabled, and
// each generator on its own enable flag, so systems without OpenTherm
// boilers (e.g. heat-pump only) get no clutter.
func (m ParamsMap) addOpenTherm(from map[string]string, static, read, write bool) {
	if !static {
		return
	}
	if from["OT_GLOBAL_ENABLE_R"] != "1" && from["OT_GLOBAL_ENABLE_A"] != "1" {
		return
	}
	temp := func(id, label string, i int) {
		m[id] = Attributes{
			Name:              "OpenTherm generator " + fmt.Sprint(i) + " " + label,
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     tplTemp16Sentinel,
		}
	}
	raw := func(id, label string, i int) {
		m[id] = Attributes{
			Name:          "OpenTherm generator " + fmt.Sprint(i) + " " + label,
			EntityType:    "sensor",
			ValueTemplate: tplRawSentinel,
		}
	}
	for i := 0; i <= 8; i++ {
		p := "OT_G" + fmt.Sprint(i)
		if from[p+"_ENABLE"] != "1" {
			continue
		}
		temp(p+"_TMAND", "flow temperature", i)
		temp(p+"_TACS", "ACS temperature", i)
		temp(p+"_TRIT", "return temperature", i)
		raw(p+"_STATUS", "status code", i)
		raw(p+"_POWER", "power (raw)", i)
		raw(p+"_ERROR", "error code", i)
		raw(p+"_OEMERROR", "OEM error code", i)
	}
}

// addGlobalOutputs exposes the physical relay outputs of the board
// (GLOBAL_OUTPUT_0..15) as diagnostic binary sensors.
func (m ParamsMap) addGlobalOutputs(from map[string]string, static, read, write bool) {
	if !static {
		return
	}
	for i := 0; i <= 15; i++ {
		id := "GLOBAL_OUTPUT_" + fmt.Sprint(i)
		v := from[id]
		// Skip missing or unconfigured (255) outputs; 0 and 1 are real states.
		if v == "" || v == "255" {
			continue
		}
		m[id] = Attributes{
			Name:          "Output " + fmt.Sprint(i),
			EntityType:    "binary_sensor",
			ValueTemplate: tplOnOff,
		}
	}
}

// addSystemAlarms exposes the global alarm flag as a problem binary sensor
// and the individual alarm words (ALARM_0..ALARM_C) as raw diagnostic
// sensors (each word is a bitfield of alarm codes).
func (m ParamsMap) addSystemAlarms(from map[string]string, static, read, write bool) {
	if !static {
		return
	}
	m["ANY_ALARM"] = Attributes{
		Name:          "Any alarm",
		EntityType:    "binary_sensor",
		DeviceClass:   "problem",
		ValueTemplate: `{% if value == "0" %}off{% else %}on{% endif %}`,
	}
	for _, s := range []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C"} {
		id := "ALARM_" + s
		if from[id] == "" {
			continue
		}
		m[id] = Attributes{
			Name:          "Alarm word " + s,
			EntityType:    "sensor",
			ValueTemplate: tplRawSentinel,
		}
	}
}

// --- Enum/select option localization -------------------------------------
//
// Home Assistant does not translate the options of MQTT select/enum entities:
// they are literal strings, used both as the dropdown choices and (for selects)
// as the command value. To offer localized dropdowns we rebuild the Options and
// the value/command templates from a per-language label table, so options and
// templates always stay in sync. English is the built-in default and is left
// untouched.

// labelSets holds the localized labels for the enum domains used by Season,
// zone/calendar forcing and calendar mode.
var labelSets = map[string]map[string]string{
	"en": {
		"winter": "winter", "summer": "summer",
		"automatic": "automatic", "off": "off", "economy": "economy", "comfort": "comfort",
		"forced_off": "forced off", "forced_economy": "forced economy", "forced_comfort": "forced comfort",
	},
	"it": {
		"winter": "inverno", "summer": "estate",
		"automatic": "automatico", "off": "spento", "economy": "economia", "comfort": "comfort",
		"forced_off": "forzato spento", "forced_economy": "forzato economia", "forced_comfort": "forzato comfort",
	},
	"de": {
		"winter": "Winter", "summer": "Sommer",
		"automatic": "automatisch", "off": "aus", "economy": "Sparbetrieb", "comfort": "Komfort",
		"forced_off": "erzwungen aus", "forced_economy": "erzwungen Sparbetrieb", "forced_comfort": "erzwungen Komfort",
	},
	"fr": {
		"winter": "hiver", "summer": "été",
		"automatic": "automatique", "off": "arrêt", "economy": "économie", "comfort": "confort",
		"forced_off": "forcé arrêt", "forced_economy": "forcé économie", "forced_comfort": "forcé confort",
	},
	"es": {
		"winter": "invierno", "summer": "verano",
		"automatic": "automático", "off": "apagado", "economy": "economía", "comfort": "confort",
		"forced_off": "forzado apagado", "forced_economy": "forzado economía", "forced_comfort": "forzado confort",
	},
}

var (
	zoneForcingRe = regexp.MustCompile(`^Z\d+_FORCING$`)
	zoneRegimeRe  = regexp.MustCompile(`^Z\d+_REGIME$`)
	calForcingRe  = regexp.MustCompile(`^MT\d+_FORCING$`)
	calModeRe     = regexp.MustCompile(`^MT\d+_MODE$`)
)

// Localize rewrites the options and value/command templates of the localizable
// enum/select entities (Season, zone forcing, calendar preset, calendar mode)
// into the given language. Unknown languages and "en" are no-ops.
func (m ParamsMap) Localize(lang string) {
	L, ok := labelSets[lang]
	if !ok || lang == "en" {
		return
	}
	for key, attr := range m {
		switch {
		case key == "GLOBAL_SEASON":
			attr.Options = []string{L["winter"], L["summer"]}
			if attr.EntityType == "select" {
				attr.ValueTemplate = fmt.Sprintf(`{%% if value == "1" %%}%s{%% else %%}%s{%% endif %%}`, L["summer"], L["winter"])
				attr.CommandTemplate = fmt.Sprintf(`{%% if value == "%s" %%}1{%% else %%}0{%% endif %%}`, L["summer"])
			} else {
				attr.ValueTemplate = fmt.Sprintf(`{%% if value == "0" %%}%s{%% elif value == "1" %%}%s{%% else %%}{{ value }}{%% endif %%}`, L["winter"], L["summer"])
			}
			m[key] = attr

		case zoneForcingRe.MatchString(key):
			// 0 automatic, 1 off, 2 economy, 3 comfort
			attr.Options = []string{L["automatic"], L["off"], L["economy"], L["comfort"]}
			if attr.EntityType == "select" {
				attr.ValueTemplate = fmt.Sprintf(`{%% if value == "1" %%}%s{%% elif value == "2" %%}%s{%% elif value == "3" %%}%s{%% else %%}%s{%% endif %%}`, L["off"], L["economy"], L["comfort"], L["automatic"])
				attr.CommandTemplate = fmt.Sprintf(`{%% if value == "%s" %%}1{%% elif value == "%s" %%}2{%% elif value == "%s" %%}3{%% else %%}0{%% endif %%}`, L["off"], L["economy"], L["comfort"])
			} else {
				attr.ValueTemplate = fmt.Sprintf(`{%% if value == "0" %%}%s{%% elif value == "1" %%}%s{%% elif value == "2" %%}%s{%% elif value == "3" %%}%s{%% else %%}{{ value }}{%% endif %%}`, L["automatic"], L["off"], L["economy"], L["comfort"])
			}
			m[key] = attr

		case zoneRegimeRe.MatchString(key):
			// From FORCING: 2 forced-eco, 3 forced-comfort, 50 auto-eco,
			// 51 auto-comfort, 1/48/49 off, else automatic.
			au := L["automatic"]
			ae := L["automatic"] + " " + L["economy"]
			ac := L["automatic"] + " " + L["comfort"]
			fe := L["forced_economy"]
			fc := L["forced_comfort"]
			attr.Options = []string{au, ae, ac, fe, fc, L["off"]}
			attr.ValueTemplate = fmt.Sprintf(`{%% if value == "2" %%}%s{%% elif value == "3" %%}%s{%% elif value == "50" %%}%s{%% elif value == "51" %%}%s{%% elif value in ["1", "48", "49"] %%}%s{%% else %%}%s{%% endif %%}`, fe, fc, ae, ac, L["off"], au)
			m[key] = attr

		case calForcingRe.MatchString(key):
			// 0 automatic, 1 forced off, 2 forced economy, 3 forced comfort
			attr.Options = []string{L["automatic"], L["forced_off"], L["forced_economy"], L["forced_comfort"]}
			if attr.EntityType == "select" {
				attr.ValueTemplate = fmt.Sprintf(`{%% if value == "1" %%}%s{%% elif value == "2" %%}%s{%% elif value == "3" %%}%s{%% else %%}%s{%% endif %%}`, L["forced_off"], L["forced_economy"], L["forced_comfort"], L["automatic"])
				attr.CommandTemplate = fmt.Sprintf(`{%% if value == "%s" %%}1{%% elif value == "%s" %%}2{%% elif value == "%s" %%}3{%% else %%}0{%% endif %%}`, L["forced_off"], L["forced_economy"], L["forced_comfort"])
			} else {
				attr.ValueTemplate = fmt.Sprintf(`{%% if value == "0" %%}%s{%% elif value == "1" %%}%s{%% elif value == "2" %%}%s{%% elif value == "3" %%}%s{%% else %%}{{ value }}{%% endif %%}`, L["automatic"], L["forced_off"], L["forced_economy"], L["forced_comfort"])
			}
			m[key] = attr

		case calModeRe.MatchString(key):
			// 1 off, 2 economy, 3 comfort
			attr.Options = []string{L["off"], L["economy"], L["comfort"]}
			attr.ValueTemplate = fmt.Sprintf(`{%% if value == "1" %%}%s{%% elif value == "2" %%}%s{%% elif value == "3" %%}%s{%% else %%}{{ value }}{%% endif %%}`, L["off"], L["economy"], L["comfort"])
			m[key] = attr
		}
	}
}
