package places

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// photonCity finds a city through Photon, for where Nominatim can't be reached (it is
// blocked from mainland China) or is turned off. Same OpenStreetMap data, so the city's
// key has the same shape: country, state and name.
func (c *Client) photonCity(ctx context.Context, q string) (*City, error) {
	if c.PhotonURL == "" {
		return nil, ErrNoCity
	}
	v := url.Values{"q": {q}, "limit": {"5"}, "lang": {"en"}}
	for _, t := range []string{"place:city", "place:town", "place:village", "place:municipality"} {
		v.Add("osm_tag", t)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, strings.TrimRight(c.PhotonURL, "/")+"/api/?"+v.Encode(), nil)
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
		return nil, fmt.Errorf("city lookup: status %d", resp.StatusCode)
	}
	var data struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				Name, State, Country string
				CountryCode          string `json:"countrycode"`
				OSMKey               string `json:"osm_key"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return nil, err
	}
	for _, f := range data.Features {
		p := f.Properties
		if p.Name == "" || p.OSMKey != "place" || len(f.Geometry.Coordinates) != 2 {
			continue
		}
		country, err := c.photonCountry(ctx, q)
		if err != nil {
			return nil, err
		}
		if country != "" && !strings.EqualFold(country, p.Name) {
			return nil, ErrNoCity
		}
		display := p.Name
		if p.State != "" && p.State != p.Name {
			display += ", " + p.State
		}
		if p.Country != "" {
			display += ", " + p.Country
		}
		return &City{Key: slug(strings.ToLower(p.CountryCode) + "-" + p.State + "-" + p.Name), Name: display,
			Lat: f.Geometry.Coordinates[1], Lon: f.Geometry.Coordinates[0]}, nil
	}
	return nil, ErrNoCity
}

// photonCountry names the country q is, if it is one: a search for "Malaysia" among towns
// otherwise settles for the nearest spelling, Mala Vyska in Ukraine.
func (c *Client) photonCountry(ctx context.Context, q string) (string, error) {
	v := url.Values{"q": {q}, "limit": {"1"}, "lang": {"en"}, "osm_tag": {"place:country"}}
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, strings.TrimRight(c.PhotonURL, "/")+"/api/?"+v.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.http().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("country lookup: status %d", resp.StatusCode)
	}
	var data struct {
		Features []struct {
			Properties struct {
				Name     string `json:"name"`
				OSMValue string `json:"osm_value"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return "", err
	}
	if len(data.Features) == 0 || data.Features[0].Properties.OSMValue != "country" {
		return "", nil
	}
	return data.Features[0].Properties.Name, nil
}
