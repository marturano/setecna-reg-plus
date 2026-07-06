package scraper

import (
	"encoding/json"
	"testing"
)

func TestExtractCSRFToken(t *testing.T) {
	cases := []struct {
		name    string
		html    string
		want    string
		wantErr bool
	}{
		{"standard order", `<html><head><meta name="csrf-token" content="abc123"></head>`, "abc123", false},
		{"reversed attributes", `<meta content="tok-42" name="csrf-token">`, "tok-42", false},
		{"single quotes", `<meta name='csrf-token' content='xyz'>`, "xyz", false},
		{"self closing", `<meta name="csrf-token" content="sc" />`, "sc", false},
		{"missing", `<html><head><title>login</title></head>`, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := extractCSRFToken([]byte(c.html))
			if (err != nil) != c.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, c.wantErr)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestFlexString(t *testing.T) {
	var r Response
	payload := `{"Status":"ok","Timestamp":"t1","Latest":"2024-01-01 10:00:00.000000+01","Rows":2,` +
		`"Data":[{"Id":"Z1_TEMP","V":215},{"Id":"GLOBAL_SEASON","V":"0"}]}`
	if err := json.Unmarshal([]byte(payload), &r); err != nil {
		t.Fatal(err)
	}
	m := r.Map()
	if m["Z1_TEMP"] != "215" {
		t.Fatalf("numeric value not normalized: %q", m["Z1_TEMP"])
	}
	if m["GLOBAL_SEASON"] != "0" {
		t.Fatalf("string value mangled: %q", m["GLOBAL_SEASON"])
	}
}
