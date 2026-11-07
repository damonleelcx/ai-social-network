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
		if r.URL.Path != "/api/" || !strings.Contains(r.URL.RawQuery, "osm_tag=place%3Acity") {
			t.Errorf("unexpected request %s", r.URL)
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
