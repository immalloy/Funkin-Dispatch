package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	userAgent     = "Funkin-Dispatch/1.0"
	defaultGameID = 8694
)

var periodLabels = map[string]string{
	"today": "Today", "week": "This Week", "month": "This Month",
	"3month": "3 Months", "6month": "6 Months", "year": "This Year",
	"alltime": "All Time",
}

type Config struct {
	GameID                     int      `json:"game_id"`
	IntervalHours              float64  `json:"interval_hours"`
	MaxPerPeriod               int      `json:"max_per_period"`
	WebhookUsername            string   `json:"webhook_username"`
	AnnounceExistingOnFirstRun bool     `json:"announce_existing_on_first_run"`
	AnnounceNewMods            bool     `json:"announce_new_mods"`
	AnnounceRankChanges        bool     `json:"announce_rank_changes"`
	AnnouncePeriodChanges      bool     `json:"announce_period_changes"`
	Periods                    []string `json:"periods"`
	Blacklist                  []string `json:"blacklist"`
	ShowFlaggedContent         bool     `json:"show_flagged_content"`
}

type Position struct {
	Period string `json:"period"`
	Rank   int    `json:"rank"`
}

type State struct {
	Version     int                    `json:"version"`
	Initialized bool                   `json:"initialized"`
	Positions   map[string]Position    `json:"positions"`
	SeenMods    map[string]interface{} `json:"seen_mods,omitempty"`
}

type Candidate struct {
	Period string
	Rank   int
	Mod    map[string]interface{}
}

type Args struct {
	Loop             bool
	IntervalHours    float64
	StateFile        string
	DryRun           bool
	AnnounceExisting bool
}

func defaultConfig() Config {
	return Config{
		GameID: defaultGameID, IntervalHours: 1, MaxPerPeriod: 3,
		WebhookUsername: "Funkin Dispatch", AnnounceNewMods: true,
		AnnounceRankChanges: true, AnnouncePeriodChanges: true,
		Periods: []string{"today"}, Blacklist: []string{},
	}
}

func loadConfig(path string) (Config, error) {
	config := defaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("read %s: %w", path, err)
	}
	if len(config.Periods) == 0 {
		return config, errors.New("config.json must enable at least one period")
	}
	if config.MaxPerPeriod < 1 {
		return config, errors.New("max_per_period must be a positive integer")
	}
	return config, nil
}

func loadState(path string) (State, error) {
	state := State{Version: 1, Positions: map[string]Position{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if err := json.Unmarshal(data, &state); err != nil {
		return state, fmt.Errorf("read %s: %w", path, err)
	}
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Positions == nil {
		state.Positions = map[string]Position{}
	}
	return state, nil
}

func saveState(state State, path string) error {
	if state.Positions == nil {
		state.Positions = map[string]Position{}
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}

func periodLabel(period string) string {
	if label, ok := periodLabels[period]; ok {
		return label
	}
	return period
}

func fetchFeaturedMods(config Config) ([]Candidate, error) {
	url := fmt.Sprintf("https://gamebanana.com/apiv13/Game/%d/TopSubs", config.GameID)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return nil, fmt.Errorf("GameBanana returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var items []map[string]interface{}
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&items); err != nil {
		return nil, err
	}
	return selectFeaturedMods(items, config), nil
}

func selectFeaturedMods(items []map[string]interface{}, config Config) []Candidate {
	blacklist := make([]string, len(config.Blacklist))
	for index, term := range config.Blacklist {
		blacklist[index] = strings.ToLower(term)
	}
	results := make([]Candidate, 0)
	seen := map[string]bool{}
	for _, period := range config.Periods {
		rank := 0
		for _, mod := range items {
			if stringValue(mod["_sPeriod"]) != period {
				continue
			}
			if !config.ShowFlaggedContent && stringValue(mod["_sInitialVisibility"]) != "show" {
				continue
			}
			name := strings.ToLower(stringValue(mod["_sName"]))
			submitter, _ := mod["_aSubmitter"].(map[string]interface{})
			author := strings.ToLower(stringValue(submitter["_sName"]))
			blocked := false
			for _, term := range blacklist {
				if strings.Contains(name, term) || strings.Contains(author, term) {
					blocked = true
					break
				}
			}
			if blocked {
				continue
			}
			modID := stringValue(mod["_idRow"])
			if modID == "" || seen[modID] {
				continue
			}
			seen[modID] = true
			rank++
			results = append(results, Candidate{Period: period, Rank: rank, Mod: mod})
			if rank >= config.MaxPerPeriod {
				break
			}
		}
	}
	return results
}

func stringValue(value interface{}) string {
	switch value := value.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	default:
		return ""
	}
}

type event int

const (
	eventNew event = iota
	eventRank
	eventPeriod
	eventDeparted
)

var eventColors = map[event]int{
	eventNew:      0x57f287,
	eventRank:     0x5865f2,
	eventPeriod:   0x9b59b6,
	eventDeparted: 0xed4245,
}

func classifyEvent(period string, previous *Position, departed bool) event {
	if departed {
		return eventDeparted
	}
	if previous == nil {
		return eventNew
	}
	if previous.Period != period {
		return eventPeriod
	}
	return eventRank
}

func eventName(kind event) string {
	switch kind {
	case eventDeparted:
		return "Mod No Longer Featured"
	case eventPeriod:
		return "Featured Period Change"
	case eventRank:
		return "Featured Ranking Update"
	default:
		return "New Mod Featured"
	}
}

func buildEmbed(period string, rank int, mod map[string]interface{}, previous *Position, departed bool, now time.Time) map[string]interface{} {
	submitter, _ := mod["_aSubmitter"].(map[string]interface{})
	modID := stringValue(mod["_idRow"])
	name := stringValue(mod["_sName"])
	if name == "" {
		if modID != "" {
			name = "Mod " + modID
		} else {
			name = "Untitled mod"
		}
	}
	modURL := stringValue(mod["_sProfileUrl"])
	if modURL == "" && modID != "" {
		modURL = "https://gamebanana.com/mods/" + modID
	}
	author := stringValue(submitter["_sName"])
	if author == "" {
		author = "Unknown creator"
	}
	authorURL := stringValue(submitter["_sProfileUrl"])
	imageURL := stringValue(mod["_sImageUrl"])
	kind := classifyEvent(period, previous, departed)
	description := ""
	if departed {
		description = fmt.Sprintf("No longer featured in **%s #%d**.", periodLabel(previous.Period), previous.Rank)
	} else if previous != nil {
		movement := fmt.Sprintf("%s #%d -> %s #%d", periodLabel(previous.Period), previous.Rank, periodLabel(period), rank)
		description = fmt.Sprintf("Moved from **%s**.", movement)
	} else {
		description = fmt.Sprintf("Now featured in **%s #%d**.", periodLabel(period), rank)
	}
	authorField := map[string]interface{}{"name": author, "url": authorURL}
	if avatar := stringValue(submitter["_sAvatarUrl"]); avatar != "" {
		authorField["icon_url"] = avatar
	}
	embed := map[string]interface{}{
		"author": authorField, "title": name, "url": modURL,
		"description": description, "color": eventColors[kind],
		"footer":    map[string]interface{}{"text": "Funkin Dispatch"},
		"timestamp": now.UTC().Format(time.RFC3339),
	}
	if imageURL != "" {
		if previous != nil && previous.Period == period {
			embed["thumbnail"] = map[string]interface{}{"url": imageURL}
		} else {
			embed["image"] = map[string]interface{}{"url": imageURL}
		}
	}
	return embed
}

func webhookBase() (string, error) {
	url := strings.TrimRight(strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL")), "/")
	if url == "" {
		return "", errors.New("DISCORD_WEBHOOK_URL is not set")
	}
	parts := strings.Split(url, "/")
	if len(parts) < 2 || parts[len(parts)-1] == "" || parts[len(parts)-2] == "" {
		return "", errors.New("DISCORD_WEBHOOK_URL does not look valid")
	}
	return "https://discord.com/api/webhooks/" + parts[len(parts)-2] + "/" + parts[len(parts)-1], nil
}

func postMod(period string, rank int, mod map[string]interface{}, config Config, previous *Position, departed bool) error {
	base, err := webhookBase()
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"username":         config.WebhookUsername,
		"embeds":           []map[string]interface{}{buildEmbed(period, rank, mod, previous, departed, time.Now())},
		"allowed_mentions": map[string]interface{}{"parse": []string{}},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, base+"?wait=true", bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return fmt.Errorf("Discord webhook returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func parseArgs() Args {
	var args Args
	flag.BoolVar(&args.Loop, "loop", false, "keep checking at the configured interval")
	flag.Float64Var(&args.IntervalHours, "interval-hours", 0, "override config.json interval")
	flag.StringVar(&args.StateFile, "state-file", "", "use a separate state file for testing")
	flag.BoolVar(&args.DryRun, "dry-run", false, "list new mods without posting or marking them seen")
	flag.BoolVar(&args.AnnounceExisting, "announce-existing", false, "announce the current feed on this run, including first-run entries")
	flag.Parse()
	return args
}

func runOnce(args Args, config Config) error {
	statePath := args.StateFile
	if statePath == "" {
		statePath = "state.json"
	}
	state, err := loadState(statePath)
	if err != nil {
		return err
	}
	candidates, err := fetchFeaturedMods(config)
	if err != nil {
		return err
	}
	fmt.Printf("[ok] fetched %d featured mod(s) across %d period(s)\n", len(candidates), len(config.Periods))
	previousPositions := state.Positions
	currentPositions := make(map[string]Position, len(candidates))
	for _, candidate := range candidates {
		currentPositions[stringValue(candidate.Mod["_idRow"])] = Position{Period: candidate.Period, Rank: candidate.Rank}
	}
	departed := make([]struct {
		id       string
		previous Position
	}, 0)
	for id, previous := range previousPositions {
		if state.Initialized {
			if _, ok := currentPositions[id]; !ok {
				departed = append(departed, struct {
					id       string
					previous Position
				}{id: id, previous: previous})
			}
		}
	}
	sort.Slice(departed, func(i, j int) bool { return departed[i].id < departed[j].id })
	changes := 0
	for _, candidate := range candidates {
		position := Position{Period: candidate.Period, Rank: candidate.Rank}
		if previous, ok := previousPositions[stringValue(candidate.Mod["_idRow"])]; !ok || previous != position {
			changes++
		}
	}
	if args.DryRun {
		for _, candidate := range candidates {
			id := stringValue(candidate.Mod["_idRow"])
			previous, ok := previousPositions[id]
			position := Position{Period: candidate.Period, Rank: candidate.Rank}
			if ok && previous == position {
				continue
			}
			if ok {
				fmt.Printf("[update] %s: %s #%d -> %s #%d\n", displayName(candidate.Mod), previous.Period, previous.Rank, candidate.Period, candidate.Rank)
			} else {
				fmt.Printf("[new] %s: %s #%d\n", displayName(candidate.Mod), candidate.Period, candidate.Rank)
			}
		}
		for _, item := range departed {
			fmt.Printf("[left] Mod %s: %s #%d\n", item.id, item.previous.Period, item.previous.Rank)
		}
		fmt.Printf("[done] found %d position change(s), %d departure(s)\n", changes, len(departed))
		return nil
	}
	if !state.Initialized && !args.AnnounceExisting && !config.AnnounceExistingOnFirstRun {
		state.Positions = currentPositions
		state.Initialized = true
		if err := saveState(state, statePath); err != nil {
			return err
		}
		fmt.Printf("[ok] initialized with %d current position(s); nothing posted\n", len(candidates))
		return nil
	}
	nextPositions := map[string]Position{}
	posted, skipped := 0, 0
	for _, item := range departed {
		mod := map[string]interface{}{"_idRow": item.id}
		if err := postMod("", 0, mod, config, &item.previous, true); err != nil {
			nextPositions[item.id] = item.previous
			fmt.Printf("[warn] failed to announce Mod %s leaving the featured feed: %v\n", item.id, err)
			continue
		}
		fmt.Printf("[ok] announced Mod %s left the featured feed\n", item.id)
		posted++
	}
	postingOrder := map[string]int{}
	for index, period := range config.Periods {
		postingOrder[period] = index
	}
	postingCandidates := append([]Candidate(nil), candidates...)
	sort.SliceStable(postingCandidates, func(i, j int) bool {
		left, right := postingCandidates[i], postingCandidates[j]
		if postingOrder[left.Period] != postingOrder[right.Period] {
			return postingOrder[left.Period] > postingOrder[right.Period]
		}
		return left.Rank > right.Rank
	})
	for _, candidate := range postingCandidates {
		id := stringValue(candidate.Mod["_idRow"])
		previous, hasPrevious := previousPositions[id]
		if !hasPrevious {
			if _, seen := state.SeenMods[id]; seen {
				nextPositions[id] = Position{Period: candidate.Period, Rank: candidate.Rank}
				continue
			}
		}
		position := Position{Period: candidate.Period, Rank: candidate.Rank}
		if hasPrevious && previous == position {
			nextPositions[id] = previous
			continue
		}
		isNew := !hasPrevious
		periodChanged := hasPrevious && previous.Period != candidate.Period
		rankChanged := hasPrevious && previous.Period == candidate.Period
		if (isNew && !config.AnnounceNewMods) || (periodChanged && !config.AnnouncePeriodChanges) || (rankChanged && !config.AnnounceRankChanges) {
			nextPositions[id] = position
			fmt.Printf("[skip] %s change disabled by config\n", displayName(candidate.Mod))
			skipped++
			continue
		}
		var previousPointer *Position
		if hasPrevious {
			previousPointer = &previous
		}
		if err := postMod(candidate.Period, candidate.Rank, candidate.Mod, config, previousPointer, false); err != nil {
			fmt.Printf("[warn] failed to announce %s: %v\n", displayName(candidate.Mod), err)
			continue
		}
		nextPositions[id] = position
		fmt.Printf("[ok] announced %s\n", displayName(candidate.Mod))
		posted++
	}
	state.Positions = nextPositions
	state.Initialized = true
	if !mapsEqual(nextPositions, previousPositions) {
		if err := saveState(state, statePath); err != nil {
			return err
		}
		fmt.Printf("[done] checked %d featured mod(s); posted %d, skipped %d (state updated)\n", len(candidates), posted, skipped)
	} else {
		fmt.Printf("[done] checked %d featured mod(s); posted %d, skipped %d (no changes)\n", len(candidates), posted, skipped)
	}
	return nil
}

func displayName(mod map[string]interface{}) string {
	if name := stringValue(mod["_sName"]); name != "" {
		return name
	}
	return "Untitled mod"
}

func mapsEqual(left, right map[string]Position) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func main() {
	args := parseArgs()
	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Printf("[fatal] %v\n", err)
		os.Exit(1)
	}
	interval := config.IntervalHours
	if args.IntervalHours != 0 {
		interval = args.IntervalHours
	}
	if interval <= 0 {
		fmt.Println("[fatal] interval must be greater than zero")
		os.Exit(1)
	}
	if !args.Loop {
		if err := runOnce(args, config); err != nil {
			fmt.Printf("[fatal] check failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	fmt.Printf("[ok] dispatch loop started; checking every %g hour(s)\n", interval)
	for {
		if err := runOnce(args, config); err != nil {
			fmt.Printf("[fatal] check failed: %v\n", err)
		}
		time.Sleep(time.Duration(interval * float64(time.Hour)))
	}
}
