package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) > 1 {
		return errors.New("usage: go run ./cmd/enrichosm [data-file]")
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
	query, sourceIDs, err := overpassQuery(places)
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
	output, err := json.MarshalIndent(places, "", "  ")
	if err != nil {
		return fmt.Errorf("encode enriched dataset: %w", err)
	}
	output = append(output, '\n')
	if err := replaceFile(path, output); err != nil {
		return fmt.Errorf("write enriched dataset: %w", err)
	}
	fmt.Printf("Added OSM details to %d of %d places (Overpass returned %d of %d IDs; data timestamp %s).\n", enriched, len(places), len(tagsByID), len(sourceIDs), updatedAt.Format(time.RFC3339))
	return nil
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
