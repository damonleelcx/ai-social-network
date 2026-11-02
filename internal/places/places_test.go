package places

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPhotonKeepsOnlyNearbyPlacesOfTheRequestedKind(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("osm_tag") != "amenity:cafe" {
			_, _ = w.Write([]byte(`{"features":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"features":[
			{"geometry":{"coordinates":[-97.74,30.27]},"properties":{"osm_id":1,"osm_type":"N","osm_key":"amenity","osm_value":"cafe","name":"Near Cafe","street":"Main St","housenumber":"5","city":"Austin"}},
			{"geometry":{"coordinates":[-96.80,32.78]},"properties":{"osm_id":2,"osm_type":"N","osm_key":"amenity","osm_value":"cafe","name":"Dallas Cafe"}},
			{"geometry":{"coordinates":[-97.74,30.27]},"properties":{"osm_id":3,"osm_type":"N","osm_key":"tourism","osm_value":"hotel","name":"A Hotel"}},
			{"geometry":{"coordinates":[-97.74,30.27]},"properties":{"osm_id":4,"osm_type":"N","osm_key":"amenity","osm_value":"cafe"}}]}`))
	}))
	defer srv.Close()
	c := &Client{PhotonURL: srv.URL}
	got := c.photonSearch(context.Background(), City{Lat: 30.27, Lon: -97.74}, []string{`"amenity"="cafe"`}, map[string]string{`"amenity"="cafe"`: "cafe"})
	if len(got) != 1 || got[0].Name != "Near Cafe" || got[0].Address != "5 Main St, Austin" || got[0].ID != "osm-n-1" {
		t.Fatalf("got %+v", got)
	}
}
