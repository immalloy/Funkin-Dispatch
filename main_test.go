package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestSelectFeaturedModsKeepsFirstPeriodAndFilters(t *testing.T) {
	config := defaultConfig()
	config.Periods = []string{"today", "week"}
	config.MaxPerPeriod = 2
	config.Blacklist = []string{"spoiler"}
	items := []map[string]interface{}{
		{"_idRow": json.Number("1"), "_sPeriod": "today", "_sInitialVisibility": "show", "_sName": "First"},
		{"_idRow": json.Number("1"), "_sPeriod": "week", "_sInitialVisibility": "show", "_sName": "First"},
		{"_idRow": json.Number("2"), "_sPeriod": "today", "_sInitialVisibility": "show", "_sName": "Spoiler mod"},
		{"_idRow": json.Number("3"), "_sPeriod": "today", "_sInitialVisibility": "hidden", "_sName": "Hidden"},
		{"_idRow": json.Number("4"), "_sPeriod": "week", "_sInitialVisibility": "show", "_sName": "Second"},
	}
	results := selectFeaturedMods(items, config)
	if len(results) != 2 || results[0].Period != "today" || results[0].Rank != 1 || stringValue(results[1].Mod["_idRow"]) != "4" {
		t.Fatalf("unexpected candidates: %#v", results)
	}
}

func TestEventColorsAndEmbedContent(t *testing.T) {
	previous := &Position{Period: "week", Rank: 2}
	mod := map[string]interface{}{
		"_idRow": json.Number("42"), "_sName": "Cool Mod",
		"_sProfileUrl": "https://gamebanana.com/mods/42",
		"_aSubmitter":  map[string]interface{}{"_sName": "Creator"},
	}
	checks := []struct {
		name   string
		kind   event
		period string
		prior  *Position
		left   bool
	}{
		{"new", eventNew, "today", nil, false},
		{"rank", eventRank, "week", &Position{Period: "week", Rank: 3}, false},
		{"period", eventPeriod, "today", previous, false},
		{"departed", eventDeparted, "", previous, true},
	}
	for _, check := range checks {
		embed := buildEmbed(check.period, 3, mod, check.prior, check.left, time.Unix(0, 0))
		if embed["color"] != eventColors[check.kind] {
			t.Errorf("%s color = %v, want %v", check.name, embed["color"], eventColors[check.kind])
		}
		if embed["title"] != "Cool Mod" || embed["timestamp"] != "1970-01-01T00:00:00Z" {
			t.Errorf("%s embed = %#v", check.name, embed)
		}
	}
}

func TestStateRoundTripPreservesOldSeenMods(t *testing.T) {
	temporary := t.TempDir() + "/state.json"
	original := State{
		Version: 1, Initialized: true,
		Positions: map[string]Position{"42": {Period: "today", Rank: 1}},
		SeenMods:  map[string]interface{}{"7": true},
	}
	if err := saveState(original, temporary); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadState(temporary)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Positions["42"] != original.Positions["42"] || loaded.SeenMods["7"] != true {
		t.Fatalf("state did not round-trip: %#v", loaded)
	}
}

func TestLoadStateAcceptsUTF8BOM(t *testing.T) {
	temporary := t.TempDir() + "/state.json"
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"initialized": true}`)...)
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := loadState(temporary)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Initialized {
		t.Fatal("BOM-prefixed state was not loaded")
	}
}

func TestLoadConfigRejectsEmptyPeriods(t *testing.T) {
	temporary := t.TempDir() + "/config.json"
	if err := writeFile(temporary, []byte(`{"periods": []}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(temporary); err == nil {
		t.Fatal("expected empty periods to fail")
	}
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}
