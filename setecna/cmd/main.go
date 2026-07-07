// Command setecna-addon bridges a Setecna REG system (via its cloud web
// interface) to Home Assistant using MQTT device-based discovery.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/Ingordigia/homeassistant-addon-setecna/models"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/discovery"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/mqtt"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/scraper"
)

type appConfig struct {
	systemID      string
	username      string
	password      string
	mqttHost      string
	mqttPort      string
	mqttUser      string
	mqttPassword  string
	advInt        bool
	readonly      bool
	diagnostics   bool
	cleanupLegacy bool
	pollInterval  time.Duration
	names         map[string]string
	activeZones   map[int]bool
}

// parseActiveZones parses the ACTIVE_ZONES env var (zone numbers separated by
// commas, spaces or newlines). Returns nil when empty, meaning "all zones".
func parseActiveZones(raw string) map[int]bool {
	sep := func(r rune) bool { return r == ',' || r == '\n' || r == ' ' || r == '\t' || r == ';' }
	m := map[int]bool{}
	for _, tok := range strings.FieldsFunc(raw, sep) {
		if n, err := strconv.Atoi(strings.TrimSpace(tok)); err == nil && n >= 1 && n <= 32 {
			m[n] = true
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// parseNames parses the ENTITY_NAMES env var: one "PREFIX=Name" (or
// "PREFIX: Name") per line. Blank lines and #-comments are ignored.
func parseNames(raw string) map[string]string {
	names := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sep := strings.IndexAny(line, "=:")
		if sep <= 0 {
			slog.Warn("ignoring malformed entity name mapping", "line", line)
			continue
		}
		key := strings.TrimSpace(line[:sep])
		val := strings.TrimSpace(line[sep+1:])
		if key != "" && val != "" {
			names[key] = val
		}
	}
	return names
}

func loadConfig() (appConfig, error) {
	cfg := appConfig{
		systemID:     os.Getenv("REG_SYSTEM_ID"),
		username:     os.Getenv("REG_USER"),
		password:     os.Getenv("REG_PASSWORD"),
		mqttHost:     os.Getenv("MQTT_HOST"),
		mqttPort:     envOr("MQTT_PORT", "1883"),
		mqttUser:     os.Getenv("MQTT_USER"),
		mqttPassword: os.Getenv("MQTT_PASSWORD"),
	}
	if cfg.systemID == "" || cfg.username == "" || cfg.password == "" {
		return cfg, errors.New("systemID, username and password are required")
	}
	if cfg.mqttHost == "" {
		return cfg, errors.New("no MQTT broker available: install the Mosquitto add-on or configure a custom broker with the mqtt_host option")
	}
	cfg.advInt = envBool("ADV_INT", false)
	cfg.readonly = envBool("READONLY", true)
	cfg.diagnostics = envBool("DIAGNOSTICS", false)
	cfg.cleanupLegacy = envBool("CLEANUP_LEGACY", true)

	seconds, err := strconv.Atoi(envOr("POLL_INTERVAL", "30"))
	if err != nil || seconds < 10 {
		seconds = 30
	}
	cfg.pollInterval = time.Duration(seconds) * time.Second
	cfg.names = parseNames(os.Getenv("ENTITY_NAMES"))
	cfg.activeZones = parseActiveZones(os.Getenv("ACTIVE_ZONES"))
	return cfg, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		slog.Info("boolean option not set, using default", "option", key, "default", def)
		return def
	}
	return v
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	slog.Info("starting Setecna add-on", "version", discovery.Version)

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
	slog.Info("service stopped")
}

func run(ctx context.Context, cfg appConfig) error {
	bridge := discovery.New(cfg.systemID, cfg.names)
	bridge.ActiveZones = cfg.activeZones
	bridge.Diagnostics = cfg.diagnostics

	// --- Setecna cloud ---------------------------------------------------
	s, err := scraper.New(cfg.systemID)
	if err != nil {
		return err
	}
	if err := retryWithBackoff(ctx, "login to Setecna servers", func() error {
		return s.Login(cfg.username, cfg.password)
	}); err != nil {
		return err
	}
	slog.Info("connected to Setecna servers", "systemID", cfg.systemID)

	// Initial full snapshot, needed to know which zones/entities exist.
	var snapshot scraper.Response
	if err := retryWithBackoff(ctx, "initial data fetch", func() error {
		var ferr error
		snapshot, ferr = s.FetchWithRelogin()
		return ferr
	}); err != nil {
		return err
	}
	responseMap := snapshot.Map()

	logNamingHelper(responseMap)

	params := make(models.ParamsMap)
	params.AddEnabledParams(responseMap, cfg.readonly)

	advClimate := cfg.advInt && !cfg.readonly
	currentSeason := responseMap["GLOBAL_SEASON"]

	// Self-update check against GitHub releases (best effort).
	latestRelease, latestReleaseURL := checkLatestRelease(ctx)
	lastReleaseCheck := time.Now()

	// stateMu guards the shared mutable state (responseMap, snapshot,
	// currentSeason, latestRelease/URL) which is read by publishAll from MQTT
	// callback goroutines and written by the main polling loop.
	var stateMu sync.Mutex

	publishAll := func(c *mqtt.Client) {
		stateMu.Lock()
		var legacy []mqtt.Message
		if cfg.cleanupLegacy {
			legacy = bridge.LegacyCleanupMessages(params)
		}
		configMsgs, err := bridge.DeviceConfigs(params, responseMap, advClimate)
		stateMsgs := bridge.StateMessages(snapshot, params)
		upMsg := bridge.UpdateStateMessage(latestRelease, latestReleaseURL)
		stateMu.Unlock()

		if err != nil {
			slog.Error("building discovery payload", "error", err)
			return
		}
		if legacy != nil {
			c.BatchPublish(legacy)
		}
		c.BatchPublish(configMsgs)
		c.BatchPublish(stateMsgs)
		c.Publish(upMsg)
		slog.Info("discovery and state published", "entities", len(params), "devices", len(configMsgs))
	}

	// --- MQTT ------------------------------------------------------------
	client, err := mqtt.Connect(mqtt.Options{
		Host:              cfg.mqttHost,
		Port:              cfg.mqttPort,
		Username:          cfg.mqttUser,
		Password:          cfg.mqttPassword,
		ClientID:          "setecna-reg-plus-" + cfg.systemID,
		AvailabilityTopic: bridge.AvailabilityTopic(),
		OnConnect: func(c *mqtt.Client) {
			// (Re)publish discovery + state and (re)subscribe on every
			// connection, including broker restarts.
			publishAll(c)

			// Republish when Home Assistant itself restarts.
			if err := c.Subscribe("homeassistant/status", 0, func(_ paho.Client, msg paho.Message) {
				if string(msg.Payload()) == "online" {
					slog.Info("Home Assistant came online, republishing discovery")
					publishAll(c)
				}
			}); err != nil {
				slog.Error("subscribing to homeassistant/status", "error", err)
			}

			if !cfg.readonly {
				if err := c.Subscribe(bridge.CommandFilter(), 0, commandHandler(s, bridge, c)); err != nil {
					slog.Error("subscribing to command topics", "error", err)
				}
			}
		},
	})
	if err != nil {
		return err
	}
	defer client.Disconnect()

	// --- Main polling loop -------------------------------------------------
	slog.Info("entering main loop", "poll_interval", cfg.pollInterval, "readonly", cfg.readonly, "advanced_integration", advClimate)

	failures := 0
	for {
		// Ask the cloud to refresh from the controller, then give it time
		// to answer before fetching the (incremental) result.
		if err := s.AskRefresh(); err != nil {
			slog.Warn("askrefresh failed", "error", err)
		}
		if !sleepCtx(ctx, cfg.pollInterval) {
			return ctx.Err()
		}

		resp, err := s.FetchWithRelogin()
		if err != nil {
			failures++
			slog.Warn("fetch failed", "error", err, "consecutive_failures", failures)
			// Exponential backoff, capped at 5 minutes.
			backoff := min(time.Duration(failures)*30*time.Second, 5*time.Minute)
			if !sleepCtx(ctx, backoff) {
				return ctx.Err()
			}
			continue
		}
		failures = 0

		client.BatchPublish(bridge.StateMessages(resp, params))

		// If the system switched between winter and summer, the climate
		// entities must be rebuilt with the seasonal setpoint topics.
		stateMu.Lock()
		for _, d := range resp.Data {
			responseMap[d.ID] = string(d.V)
		}
		season := responseMap["GLOBAL_SEASON"]
		seasonChanged := season != currentSeason && advClimate
		if seasonChanged {
			currentSeason = season
			snapshot = resp
		}
		stateMu.Unlock()
		if seasonChanged {
			slog.Info("season changed, republishing climate discovery", "season", season)
			publishAll(client)
		}

		// Re-check for a newer add-on release once a day.
		if time.Since(lastReleaseCheck) > 24*time.Hour {
			lastReleaseCheck = time.Now()
			if v, u := checkLatestRelease(ctx); v != "" {
				stateMu.Lock()
				latestRelease, latestReleaseURL = v, u
				upMsg := bridge.UpdateStateMessage(v, u)
				stateMu.Unlock()
				client.Publish(upMsg)
			}
		}
	}
}

// logNamingHelper prints the custom labels stored in the Setecna system and
// the description code of each active zone/circuit, to help the user fill in
// the entity_names option. It does not attempt to auto-map codes to labels,
// since the built-in description dictionary is not reverse engineered.
func logNamingHelper(rm map[string]string) {
	var labels []string
	for i := 1; i <= 16; i++ {
		if v := strings.TrimSpace(rm["_FREEDESC"+strconv.Itoa(i)]); v != "" {
			labels = append(labels, fmt.Sprintf("_FREEDESC%d=%q", i, v))
		}
	}
	for i := 1; i <= 48; i++ {
		if v := strings.TrimSpace(rm["_XFREEDESC"+strconv.Itoa(i)]); v != "" {
			labels = append(labels, fmt.Sprintf("_XFREEDESC%d=%q", i, v))
		}
	}
	if len(labels) > 0 {
		slog.Info("Setecna custom labels (use them to fill the entity_names option)",
			"labels", strings.Join(labels, "  "))
	}
	var zones []string
	for i := 1; i <= 32; i++ {
		z := "Z" + strconv.Itoa(i)
		if rm[z+"_SENSOR_CHN"] != "0" && rm[z+"_SENSOR_CHN"] != "" {
			zones = append(zones, fmt.Sprintf("%s(descr=%s)", z, rm[z+"_DESCR"]))
		}
	}
	if len(zones) > 0 {
		slog.Info("active zones and their Setecna description code", "zones", strings.Join(zones, " "))
	}
}

// checkLatestRelease queries the GitHub releases API for the latest tag.
// It is best effort: on any error it returns empty strings and the update
// entity simply keeps reporting the running version.
func checkLatestRelease(ctx context.Context) (version, url string) {
	// REBRAND: if you fork this under a different GitHub owner/repo name,
	// update githubRepo (and repoURL in pkg/discovery) to match.
	const githubRepo = "marturano/setecna-reg-plus"
	const api = "https://api.github.com/repos/" + githubRepo + "/releases/latest"

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, api, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Debug("release check failed", "error", err)
		return "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	var rel struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", ""
	}
	return strings.TrimPrefix(rel.TagName, "v"), rel.HTMLURL
}

// commandHandler pushes values written by Home Assistant back to Setecna.
func commandHandler(s *scraper.Scraper, bridge *discovery.Bridge, c *mqtt.Client) paho.MessageHandler {
	return func(_ paho.Client, msg paho.Message) {
		// Topic format: setecna/<systemID>/<PARAM>/set
		parts := strings.Split(msg.Topic(), "/")
		if len(parts) != 4 || parts[3] != "set" {
			slog.Warn("ignoring message on unexpected topic", "topic", msg.Topic())
			return
		}
		param, value := parts[2], string(msg.Payload())
		slog.Info("command received", "param", param, "value", value)
		if err := s.Push(param, value); err != nil {
			slog.Error("pushing value to Setecna failed", "param", param, "error", err)
			return
		}
		// Optimistically echo the new value on the state topic so Home
		// Assistant reflects it immediately, without waiting for the next
		// poll cycle. Writes to global params use the "P_" prefix, but their
		// state is published under the unprefixed name, so strip it here.
		if err := c.Publish(mqtt.Message{
			Topic:   bridge.StateTopic(strings.TrimPrefix(param, "P_")),
			Payload: value,
			Qos:     0,
			Retain:  true,
		}); err != nil {
			slog.Warn("optimistic echo failed", "param", param, "error", err)
		}
	}
}

// retryWithBackoff retries op until it succeeds or ctx is cancelled.
func retryWithBackoff(ctx context.Context, what string, op func() error) error {
	delay := 15 * time.Second
	for {
		err := op()
		if err == nil {
			return nil
		}
		slog.Warn(what+" failed, retrying", "error", err, "retry_in", delay)
		if !sleepCtx(ctx, delay) {
			return ctx.Err()
		}
		delay = min(delay*2, 5*time.Minute)
	}
}

// sleepCtx sleeps for d, returning false if ctx was cancelled first.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
