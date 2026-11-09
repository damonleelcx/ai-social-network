package places

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPhotonCity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/" || !strings.Contains(r.URL.RawQuery, "osm_tag=place%3A") {
			t.Errorf("unexpected request %s", r.URL)
		}
		if strings.Contains(r.URL.RawQuery, "osm_tag=place%3Acountry") {
			_, _ = w.Write([]byte(`{"features":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"features":[
			{"geometry":{"coordinates":[121.3,30.3]},"properties":{"name":"Qianwan","osm_key":"highway","countrycode":"CN"}},
			{"geometry":{"coordinates":[120.2052342,30.2489634]},"properties":{"name":"Hangzhou","state":"Zhejiang","country":"China","countrycode":"CN","osm_key":"place"}}]}`))
	}))
	defer srv.Close()
	c := &Client{PhotonURL: srv.URL}
	city, err := c.photonCity(context.Background(), "杭州")
	if err != nil {
		t.Fatal(err)
	}
	if city.Key != "cn-zhejiang-hangzhou" || city.Name != "Hangzhou, Zhejiang, China" || city.Lat < 30 || city.Lon < 120 {
		t.Fatalf("got %+v", city)
	}
	if _, err := (&Client{}).photonCity(context.Background(), "x"); err != ErrNoCity {
		t.Fatalf("no Photon configured: %v", err)
	}
}

// A country is not a city, however close a town's spelling comes.
func TestPhotonCountryIsNotACity(t *testing.T) {
	country := "Malaysia"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "osm_tag=place%3Acountry") {
			_, _ = w.Write([]byte(`{"features":[{"properties":{"name":"` + country + `","osm_value":"country"}}]}`))
			return
		}
		name := "Mala Vyska"
		if country == "Singapore" {
			name = "Singapore"
		}
		_, _ = w.Write([]byte(`{"features":[{"geometry":{"coordinates":[31.6,48.6]},"properties":{"name":"` + name + `","countrycode":"XX","osm_key":"place"}}]}`))
	}))
	defer srv.Close()
	c := &Client{PhotonURL: srv.URL}
	if _, err := c.photonCity(context.Background(), "Malaysia"); err != ErrNoCity {
		t.Fatalf("a country resolved to a town: %v", err)
	}
	country = "Singapore"
	if city, err := c.photonCity(context.Background(), "Singapore"); err != nil || city.Name != "Singapore" {
		t.Fatalf("a city that is its own country: %+v %v", city, err)
	}
}
