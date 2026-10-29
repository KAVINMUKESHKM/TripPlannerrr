package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultDataset = "data/tamil_nadu_attractions.json"
const defaultOverpassURL = "https://overpass.private.coffee/api/interpreter"

type attraction struct {
	SourceID          string     `json:"source_id"`
	SourceURL         string     `json:"source_url"`
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	Category          string     `json:"category"`
	Latitude          float64    `json:"latitude"`
	Longitude         float64    `json:"longitude"`
	Address           string     `json:"address,omitempty"`
	OpeningHours      string     `json:"opening_hours,omitempty"`
	Wheelchair        string     `json:"wheelchair,omitempty"`
	Website           string     `json:"website,omitempty"`
	Phone             string     `json:"phone,omitempty"`
	Fee               string     `json:"fee,omitempty"`
	Toilets           string     `json:"toilets,omitempty"`
	ToiletsWheelchair string     `json:"toilets_wheelchair,omitempty"`
	InternetAccess    string     `json:"internet_access,omitempty"`
	Wikipedia         string     `json:"wikipedia,omitempty"`
	Wikidata          string     `json:"wikidata,omitempty"`
	DetailsUpdatedAt  *time.Time `json:"details_updated_at,omitempty"`
}

type overpassResponse struct {
	OSM3S struct {
		Timestamp string `json:"timestamp_osm_base"`
	} `json:"osm3s"`
	Elements []struct {
		Type string            `json:"type"`
		ID   int64             `json:"id"`
		Tags map[string]string `json:"tags"`
	} `json:"elements"`
}

type wikidataEntity struct {
	Descriptions map[string]struct {
		Value string `json:"value"`
	} `json:"descriptions"`
	Claims map[string][]struct {
		Mainsnak struct {
			Datavalue struct {
				Value json.RawMessage `json:"value"`
			} `json:"datavalue"`
		} `json:"mainsnak"`
	} `json:"claims"`
}

type wikidataResponse struct {
	Entities map[string]wikidataEntity `json:"entities"`
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	wikidataOnly := len(args) > 0 && args[0] == "--wikidata-only"
	if wikidataOnly {
		args = args[1:]
	}
	if len(args) > 1 {
		return errors.New("usage: go run ./cmd/enrichosm [--wikidata-only] [data-file]")
	}
	path := defaultDataset
	if len(args) == 1 {
		path = args[0]
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read attraction dataset: %w", err)
	}
	var places []attraction
	if err := json.Unmarshal(data, &places); err != nil {
		return fmt.Errorf("decode attraction dataset: %w", err)
	}
	if len(places) == 0 {
		return errors.New("attraction dataset is empty")
	}
	if wikidataOnly {
		return enrichWikidataOnly(ctx, path, places)
	}
	query, _, err := overpassQuery(places)
	if err != nil {
		return err
	}
	endpoint := strings.TrimSpace(os.Getenv("OVERPASS_URL"))
	if endpoint == "" {
		endpoint = defaultOverpassURL
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, strings.NewReader(url.Values{"data": {query}}.Encode()))
	if err != nil {
		return fmt.Errorf("create Overpass request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "TripPlanner place-details importer")
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request OpenStreetMap details: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Overpass returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return fmt.Errorf("read Overpass response: %w", err)
	}
	var result overpassResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode Overpass response: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339, result.OSM3S.Timestamp)
	if err != nil {
		updatedAt = time.Now().UTC()
	}
	tagsByID := make(map[string]map[string]string, len(result.Elements))
	for _, element := range result.Elements {
		key := element.Type + "/" + strconv.FormatInt(element.ID, 10)
		tagsByID[key] = element.Tags
	}
	enriched := 0
	for i := range places {
		tags := tagsByID[places[i].SourceID]
		if len(tags) == 0 {
			continue
		}
		places[i].Address = addressFromTags(tags)
		places[i].OpeningHours = tags["opening_hours"]
		places[i].Wheelchair = tags["wheelchair"]
		places[i].Website = firstTag(tags, "website", "contact:website", "url")
		places[i].Phone = firstTag(tags, "phone", "contact:phone")
		places[i].Fee = tags["fee"]
		places[i].Toilets = tags["toilets"]
		places[i].ToiletsWheelchair = tags["toilets:wheelchair"]
		places[i].InternetAccess = tags["internet_access"]
		places[i].Wikipedia = tags["wikipedia"]
		places[i].Wikidata = tags["wikidata"]
		if places[i].Description == "" {
			places[i].Description = tags["description"]
		}
		places[i].DetailsUpdatedAt = &updatedAt
		enriched++
	}
	wikidataDetails, err := fetchWikidataDetails(requestCtx, client, places)
	if err != nil {
		log.Printf("Wikidata enrichment unavailable; keeping OpenStreetMap details: %v", err)
	}
	for i := range places {
		details, ok := wikidataDetails[places[i].Wikidata]
		if !ok {
			continue
		}
		if places[i].Description == "" {
			places[i].Description = details.Description
		}
		if places[i].Website == "" {
			places[i].Website = details.Website
		}
	}
	if err := writeDataset(path, places); err != nil {
		return err
	}
	withDescription := 0
	for _, place := range places {
		if place.Description != "" {
			withDescription++
		}
	}
	fmt.Printf("Updated %d OSM place records; added Wikidata details where linked (%d of %d now have descriptions; Overpass timestamp %s).\n", enriched, withDescription, len(places), updatedAt.Format(time.RFC3339))
	return nil
}

func enrichWikidataOnly(ctx context.Context, path string, places []attraction) error {
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	details, err := fetchWikidataDetails(requestCtx, &http.Client{Timeout: 25 * time.Second}, places)
	if err != nil {
		return err
	}
	withDescription, withWebsite := 0, 0
	for i := range places {
		item, ok := details[places[i].Wikidata]
		if ok {
			if places[i].Description == "" {
				places[i].Description = item.Description
			}
			if places[i].Website == "" {
				places[i].Website = item.Website
			}
		}
		if places[i].Description != "" {
			withDescription++
		}
		if places[i].Website != "" {
			withWebsite++
		}
	}
	if err := writeDataset(path, places); err != nil {
		return err
	}
	fmt.Printf("Added Wikidata descriptions and websites where available (%d places have descriptions; %d have websites).\n", withDescription, withWebsite)
	return nil
}

func writeDataset(path string, places []attraction) error {
	output, err := json.MarshalIndent(places, "", "  ")
	if err != nil {
		return fmt.Errorf("encode enriched dataset: %w", err)
	}
	output = append(output, '\n')
	if err := replaceFile(path, output); err != nil {
		return fmt.Errorf("write enriched dataset: %w", err)
	}
	return nil
}

type placeDetails struct {
	Description string
	Website     string
}

func fetchWikidataDetails(ctx context.Context, client *http.Client, places []attraction) (map[string]placeDetails, error) {
	unique := make(map[string]struct{})
	for _, place := range places {
		if len(place.Wikidata) > 1 && place.Wikidata[0] == 'Q' {
			if _, err := strconv.ParseInt(place.Wikidata[1:], 10, 64); err == nil {
				unique[place.Wikidata] = struct{}{}
			}
		}
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	results := make(map[string]placeDetails, len(ids))
	for start := 0; start < len(ids); start += 50 {
		end := min(start+50, len(ids))
		query := url.Values{
			"action":    {"wbgetentities"},
			"ids":       {strings.Join(ids[start:end], "|")},
			"props":     {"descriptions|claims"},
			"languages": {"en"},
			"format":    {"json"},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.wikidata.org/w/api.php?"+query.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("create Wikidata request: %w", err)
		}
		req.Header.Set("User-Agent", "TripPlanner place-details importer (OpenStreetMap/Wikidata enrichment)")
		response, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request Wikidata details: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 8<<20))
		_ = response.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read Wikidata response: %w", readErr)
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Wikidata returned HTTP %d", response.StatusCode)
		}
		var payload wikidataResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("decode Wikidata response: %w", err)
		}
		for id, entity := range payload.Entities {
			details := placeDetails{Description: entity.Descriptions["en"].Value}
			if claims := entity.Claims["P856"]; len(claims) > 0 {
				_ = json.Unmarshal(claims[0].Mainsnak.Datavalue.Value, &details.Website)
				details.Website = strings.TrimSpace(details.Website)
			}
			results[id] = details
		}
	}
	return results, nil
}

func overpassQuery(places []attraction) (string, map[string]struct{}, error) {
	ids := map[string]map[int64]struct{}{"node": {}, "way": {}, "relation": {}}
	lookup := make(map[string]struct{}, len(places))
	for _, place := range places {
		parts := strings.SplitN(place.SourceID, "/", 2)
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid OpenStreetMap source ID %q", place.SourceID)
		}
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || id < 1 {
			return "", nil, fmt.Errorf("invalid OpenStreetMap source ID %q", place.SourceID)
		}
		if _, ok := ids[parts[0]]; !ok {
			return "", nil, fmt.Errorf("unsupported OpenStreetMap source type %q", parts[0])
		}
		ids[parts[0]][id] = struct{}{}
		lookup[place.SourceID] = struct{}{}
	}
	var query strings.Builder
	query.WriteString("[out:json][timeout:90];\n(\n")
	for _, kind := range []string{"node", "way", "relation"} {
		if len(ids[kind]) == 0 {
			continue
		}
		values := make([]string, 0, len(ids[kind]))
		for id := range ids[kind] {
			values = append(values, strconv.FormatInt(id, 10))
		}
		query.WriteString(kind + "(id:" + strings.Join(values, ",") + ");\n")
	}
	query.WriteString(");\nout tags;")
	return query.String(), lookup, nil
}

func addressFromTags(tags map[string]string) string {
	if value := strings.TrimSpace(tags["addr:full"]); value != "" {
		return value
	}
	line := strings.TrimSpace(strings.TrimSpace(tags["addr:housenumber"]) + " " + strings.TrimSpace(tags["addr:street"]))
	locality := strings.TrimSpace(tags["addr:city"])
	if locality == "" {
		locality = strings.TrimSpace(tags["addr:town"])
	}
	if locality == "" {
		locality = strings.TrimSpace(tags["addr:village"])
	}
	postcode := strings.TrimSpace(tags["addr:postcode"])
	if postcode != "" {
		locality = strings.TrimSpace(locality + " " + postcode)
	}
	parts := make([]string, 0, 2)
	if line != "" {
		parts = append(parts, line)
	}
	if locality != "" {
		parts = append(parts, locality)
	}
	return strings.Join(parts, ", ")
}

func firstTag(tags map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(tags[key]); value != "" {
			return value
		}
	}
	return ""
}

func replaceFile(path string, content []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".enriched-osm-*.json")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(temporary.Name())
	}()
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}
