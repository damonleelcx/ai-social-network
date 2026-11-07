// Package places answers two questions from OpenStreetMap: "which city is this?"
// and "what public places near there suit these two people?". Results are cached
// in Postgres, which is also how we respect the public servers' usage policies.
package places

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type City struct {
	Key  string  `json:"key"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type Venue struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Kind     string  `json:"kind"`
	Address  string  `json:"address"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Website  string  `json:"website,omitempty"`
	Hours    string  `json:"hours,omitempty"`
	Reason   string  `json:"reason,omitempty"`
	Matched  string  `json:"matched,omitempty"`
}

type Client struct {
	Pool         *pgxpool.Pool
	NominatimURL string
	OverpassURL  string
	// PhotonURL is tried first, being much faster; Overpass is the fallback.
	PhotonURL string
	HTTP      *http.Client
}

const userAgent = "SynAgent/1.0 (+https://match.heros-agent.space; support@heros-agent.space)"

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 45 * time.Second}
}

func (c *Client) cached(ctx context.Context, key string, ttl time.Duration, into any) bool {
	var raw []byte
	err := c.Pool.QueryRow(ctx, `SELECT data FROM cached_lookups WHERE key=$1 AND fetched_at > now() - $2::interval`, key, fmt.Sprintf("%d seconds", int(ttl.Seconds()))).Scan(&raw)
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, into) == nil
}

func (c *Client) store(ctx context.Context, key string, v any) {
	raw, _ := json.Marshal(v)
	_, _ = c.Pool.Exec(ctx, `INSERT INTO cached_lookups (key, data) VALUES ($1,$2)
		ON CONFLICT (key) DO UPDATE SET data=EXCLUDED.data, fetched_at=now()`, key, raw)
}

var ErrNoCity = errors.New("no such city found")

var reSlug = regexp.MustCompile(`[^\p{L}\p{N}]+`)

func slug(s string) string { return strings.Trim(strings.ToLower(reSlug.ReplaceAllString(s, "-")), "-") }

// ResolveCity turns free text into a canonical city. The key is what matching
// compares, so "NYC" and "New York" must land on one key.
func (c *Client) ResolveCity(ctx context.Context, q string) (*City, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, ErrNoCity
	}
	cacheKey := "city:" + strings.ToLower(q)
	var city City
	if c.cached(ctx, cacheKey, 90*24*time.Hour, &city) {
		return &city, nil
	}
	got, err := c.nominatimCity(ctx, q)
	if err != nil && err != ErrNoCity || got == nil && c.NominatimURL == "" {
		// Nominatim is off or unreachable (it is blocked from mainland China): try Photon.
		if alt, perr := c.photonCity(ctx, q); perr == nil {
			got, err = alt, nil
		} else if err == nil {
			err = perr
		}
	}
	if err != nil {
		return nil, err
	}
	c.store(ctx, cacheKey, *got)
	return got, nil
}

func (c *Client) nominatimCity(ctx context.Context, q string) (*City, error) {
	if c.NominatimURL == "" {
		return nil, nil
	}
	var city City
	u := c.NominatimURL + "/search?format=jsonv2&addressdetails=1&limit=5&accept-language=en&featureType=settlement&q=" + url.QueryEscape(q)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("city lookup: status %d", resp.StatusCode)
	}
	var hits []struct {
		Lat     string `json:"lat"`
		Lon     string `json:"lon"`
		Address struct {
			City, Town, Village, Municipality, State, Country string
			CountryCode                                       string `json:"country_code"`
		} `json:"address"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&hits); err != nil {
		return nil, err
	}
	for _, h := range hits {
		name := first(h.Address.City, h.Address.Town, h.Address.Municipality, h.Address.Village)
		if name == "" {
			continue
		}
		var lat, lon float64
		fmt.Sscan(h.Lat, &lat)
		fmt.Sscan(h.Lon, &lon)
		display := name
		if h.Address.State != "" {
			display += ", " + h.Address.State
		}
		if h.Address.Country != "" {
			display += ", " + h.Address.Country
		}
		city = City{Key: slug(h.Address.CountryCode + "-" + h.Address.State + "-" + name), Name: display, Lat: lat, Lon: lon}
		return &city, nil
	}
	return nil, ErrNoCity
}

func first(l ...string) string {
	for _, s := range l {
		if s != "" {
			return s
		}
	}
	return ""
}

// osmTags maps interest stems to OpenStreetMap tags. Every tag here is a public
// place: a first date never happens at someone's home, hotel or private club.
var osmTags = []struct {
	stems []string
	tag   string
	kind  string
}{
	{[]string{"coffee", "tea", "brunch", "read", "book"}, `"amenity"="cafe"`, "cafe"},
	{[]string{"book", "read", "literature", "poetry"}, `"shop"="books"`, "bookshop"},
	{[]string{"wine", "cocktail", "beer", "jazz", "music"}, `"amenity"="bar"`, "bar"},
	{[]string{"food", "ramen", "sushi", "cook", "restaurant"}, `"amenity"="restaurant"`, "restaurant"},
	{[]string{"art", "museum", "history", "design", "paint", "photograph"}, `"tourism"="museum"`, "museum"},
	{[]string{"art", "gallery", "paint", "photograph", "design"}, `"tourism"="gallery"`, "gallery"},
	{[]string{"film", "movie", "cinema"}, `"amenity"="cinema"`, "cinema"},
	{[]string{"theat", "comedy", "opera"}, `"amenity"="theatre"`, "theatre"},
	{[]string{"jazz", "concert", "music", "dj", "dance", "dancing"}, `"amenity"="music_venue"`, "music venue"},
	{[]string{"hik", "run", "nature", "walk", "dog", "picnic", "yoga", "cycl", "garden"}, `"leisure"="park"`, "park"},
	{[]string{"garden", "nature", "plant"}, `"leisure"="garden"`, "garden"},
	{[]string{"climb", "boulder"}, `"sport"="climbing"`, "climbing gym"},
	{[]string{"board game", "game", "chess", "dnd"}, `"shop"="games"`, "game shop"},
	{[]string{"bowling"}, `"leisure"="bowling_alley"`, "bowling"},
}

// generic fills the list when interests name nothing we can map.
var generic = []struct{ tag, kind string }{{`"amenity"="cafe"`, "cafe"}, {`"leisure"="park"`, "park"}, {`"tourism"="museum"`, "museum"}}

func tagsFor(interests []string) (tags []string, kinds map[string]string, matched map[string]string) {
	seen := map[string]bool{}
	kinds, matched = map[string]string{}, map[string]string{}
	for _, in := range interests {
		l := strings.ToLower(in)
		for _, t := range osmTags {
			for _, st := range t.stems {
				if strings.Contains(l, st) {
					if !seen[t.tag] {
						seen[t.tag] = true
						tags = append(tags, t.tag)
						kinds[t.tag] = t.kind
						matched[t.kind] = in
					}
					break
				}
			}
		}
	}
	for _, g := range generic {
		if !seen[g.tag] {
			seen[g.tag] = true
			tags = append(tags, g.tag)
			kinds[g.tag] = g.kind
		}
	}
	return tags, kinds, matched
}

// PublicKinds lists every venue kind the search can return. The date-plan verifier
// checks a shortlist against it.
func PublicKinds() map[string]bool {
	out := map[string]bool{}
	for _, t := range osmTags {
		out[t.kind] = true
	}
	for _, g := range generic {
		out[g.kind] = true
	}
	return out
}

// Search finds named public places around the city center, balanced across kinds
// so one kind (there are always many cafes) doesn't crowd out the rest.
func (c *Client) Search(ctx context.Context, city City, interests []string, limit int) ([]Venue, error) {
	tags, kinds, matched := tagsFor(interests)
	sort.Strings(tags)
	cacheKey := "venues:" + city.Key + ":" + strings.Join(tags, ",")
	var all []Venue
	cached := c.cached(ctx, cacheKey, 14*24*time.Hour, &all)
	if !cached && c.PhotonURL != "" {
		// Too few results means the service is struggling, not that the city has none.
		if found := c.photonSearch(ctx, city, tags, kinds); len(found) >= 3 {
			all = found
			cached = true
			c.store(ctx, cacheKey, all)
		}
	}
	if !cached {
		var q strings.Builder
		q.WriteString("[out:json][timeout:30];(")
		for _, t := range tags {
			fmt.Fprintf(&q, `nwr[%s]["name"](around:6000,%f,%f);`, t, city.Lat, city.Lon)
		}
		q.WriteString(");out center tags 400;")
		resp, err := c.overpass(ctx, q.String())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var data struct {
			Elements []struct {
				Type   string            `json:"type"`
				ID     int64             `json:"id"`
				Lat    float64           `json:"lat"`
				Lon    float64           `json:"lon"`
				Center *struct{ Lat, Lon float64 } `json:"center"`
				Tags   map[string]string `json:"tags"`
			} `json:"elements"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&data); err != nil {
			return nil, err
		}
		for _, e := range data.Elements {
			kind := kindOf(e.Tags, kinds)
			if kind == "" || e.Tags["name"] == "" || e.Tags["access"] == "private" {
				continue
			}
			lat, lon := e.Lat, e.Lon
			if e.Center != nil {
				lat, lon = e.Center.Lat, e.Center.Lon
			}
			all = append(all, Venue{ID: fmt.Sprintf("osm-%s-%d", e.Type, e.ID), Name: e.Tags["name"], Kind: kind, Address: address(e.Tags), Lat: lat, Lon: lon,
				Website: first(e.Tags["website"], e.Tags["contact:website"]), Hours: e.Tags["opening_hours"]})
		}
		c.store(ctx, cacheKey, all)
	}
	return balance(all, matched, limit), nil
}

// overpass tries each configured server in turn. The public ones are volunteer-run and
// often at capacity, so one being busy shouldn't cost anyone their date ideas.
func (c *Client) overpass(ctx context.Context, query string) (*http.Response, error) {
	var last error
	for _, base := range strings.Split(c.OverpassURL, ",") {
		// Each server gets its own slice of the time, so a hung one can't use it all.
		reqCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, strings.TrimSpace(base), strings.NewReader(url.Values{"data": {query}}.Encode()))
		if err != nil {
			cancel()
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", userAgent)
		resp, err := c.http().Do(req)
		if err == nil && resp.StatusCode == 200 {
			resp.Body = cancelOnClose{resp.Body, cancel}
			return resp, nil
		}
		if err == nil {
			resp.Body.Close()
			err = fmt.Errorf("status %d from %s", resp.StatusCode, base)
		}
		cancel()
		last = err
		slog.Warn("venue search server failed", "err", err)
		if ctx.Err() != nil {
			break
		}
	}
	return nil, fmt.Errorf("venue search: %w", last)
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c cancelOnClose) Close() error {
	defer c.cancel()
	return c.ReadCloser.Close()
}

func kindOf(tags map[string]string, kinds map[string]string) string {
	for tag, kind := range kinds {
		parts := strings.SplitN(strings.ReplaceAll(tag, `"`, ""), "=", 2)
		if tags[parts[0]] == parts[1] {
			return kind
		}
	}
	return ""
}

func address(t map[string]string) string {
	s := strings.TrimSpace(t["addr:housenumber"] + " " + t["addr:street"])
	if city := t["addr:city"]; city != "" {
		if s != "" {
			s += ", "
		}
		s += city
	}
	return s
}

// balance prefers kinds tied to a shared interest, then takes turns across kinds,
// and within a kind prefers places with an address and a website.
func balance(all []Venue, matched map[string]string, limit int) []Venue {
	by := map[string][]Venue{}
	for _, v := range all {
		v.Matched = matched[v.Kind]
		by[v.Kind] = append(by[v.Kind], v)
	}
	order := make([]string, 0, len(by))
	for k := range by {
		order = append(order, k)
		l := by[k]
		sort.SliceStable(l, func(i, j int) bool { return richness(l[i]) > richness(l[j]) })
	}
	sort.Slice(order, func(i, j int) bool {
		mi, mj := matched[order[i]] != "", matched[order[j]] != ""
		if mi != mj {
			return mi
		}
		return order[i] < order[j]
	})
	out := []Venue{}
	for round := 0; len(out) < limit; round++ {
		added := false
		for _, k := range order {
			if round < len(by[k]) && len(out) < limit {
				out = append(out, by[k][round])
				added = true
			}
		}
		if !added {
			break
		}
	}
	return out
}

func richness(v Venue) int {
	n := 0
	if v.Address != "" {
		n += 2
	}
	if v.Website != "" {
		n++
	}
	if v.Hours != "" {
		n++
	}
	return n
}

