package places

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// photonSearch asks Photon, a fast search over the same OpenStreetMap data, for each
// kind of place in turn. It takes a keyword plus a tag filter, and returns nothing
// outside the tag, so the public-places-only rule still holds.
func (c *Client) photonSearch(ctx context.Context, city City, tags []string, kinds map[string]string) []Venue {
	seen := map[string]bool{}
	out := []Venue{}
	for _, tag := range tags {
		parts := strings.SplitN(strings.ReplaceAll(tag, `"`, ""), "=", 2)
		if len(parts) != 2 || parts[0] == "sport" {
			continue // Photon filters on the main class of a place, which sport isn't
		}
		list, err := c.photonOne(ctx, city, parts[0], parts[1], kinds[tag])
		if err != nil {
			slog.Warn("photon search failed", "tag", tag, "err", err)
			continue
		}
		for _, v := range list {
			if !seen[v.ID] {
				seen[v.ID] = true
				out = append(out, v)
			}
		}
	}
	return out
}

func (c *Client) photonOne(ctx context.Context, city City, key, value, kind string) ([]Venue, error) {
	q := url.Values{"q": {strings.ReplaceAll(value, "_", " ")}, "lat": {fmt.Sprint(city.Lat)}, "lon": {fmt.Sprint(city.Lon)},
		"limit": {"20"}, "lang": {"en"}, "osm_tag": {key + ":" + value}}
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, strings.TrimRight(c.PhotonURL, "/")+"/api/?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var data struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				OSMID       int64  `json:"osm_id"`
				OSMType     string `json:"osm_type"`
				OSMKey      string `json:"osm_key"`
				OSMValue    string `json:"osm_value"`
				Name        string `json:"name"`
				Street      string `json:"street"`
				HouseNumber string `json:"housenumber"`
				City        string `json:"city"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&data); err != nil {
		return nil, err
	}
	out := []Venue{}
	for _, f := range data.Features {
		p := f.Properties
		if p.Name == "" || p.OSMKey != key || p.OSMValue != value || len(f.Geometry.Coordinates) != 2 {
			continue
		}
		lon, lat := f.Geometry.Coordinates[0], f.Geometry.Coordinates[1]
		// The location only biases Photon's ranking, so distance is checked here.
		if km(city.Lat, city.Lon, lat, lon) > 10 {
			continue
		}
		addr := strings.TrimSpace(p.HouseNumber + " " + p.Street)
		if p.City != "" {
			if addr != "" {
				addr += ", "
			}
			addr += p.City
		}
		out = append(out, Venue{ID: fmt.Sprintf("osm-%s-%d", strings.ToLower(p.OSMType), p.OSMID), Name: p.Name, Kind: kind, Address: addr, Lat: lat, Lon: lon})
	}
	return out, nil
}

func km(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371
	rad := math.Pi / 180
	dLat, dLon := (lat2-lat1)*rad, (lon2-lon1)*rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * r * math.Asin(math.Sqrt(a))
}
