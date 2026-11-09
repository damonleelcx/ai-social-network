package places

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Replies as Nominatim really gave them.
func TestNominatimCity(t *testing.T) {
	const (
		malaysia  = `[{"addresstype":"country","address":{"country":"Malaysia","country_code":"my"}}]`
		singapore = `[{"addresstype":"country","address":{"country":"Singapore","country_code":"sg"}}]`
		marais    = `[{"lat":"48.86","lon":"2.36","address":{"city":"Paris","state":"Ile-de-France","country":"France","country_code":"fr"}}]`
		sgCity    = `[{"lat":"1.29","lon":"103.85","address":{"city":"Singapore","country":"Singapore","country_code":"sg"}}]`
		kl        = `[{"lat":"3.15","lon":"101.69","extratags":{"linked_place":"city"},"address":{"state":"Kuala Lumpur","country":"Malaysia","country_code":"my"}}]`
		ca        = `[{"lat":"36.7","lon":"-118.7","extratags":{},"address":{"state":"California","country":"United States","country_code":"us"}}]`
		none      = `[]`
	)
	cases := []struct {
		q, country, settlement string
		want                   string // the city's name, or "" for no city
	}{
		{"马来西亚", malaysia, marais, ""},
		{"Singapore", singapore, sgCity, "Singapore, Singapore"},
		{"吉隆坡", none, kl, "Kuala Lumpur, Malaysia"},
		{"California", none, ca, ""},
		{"Paris, France", none, marais, "Paris, Ile-de-France, France"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("featureType") == "country" {
				_, _ = w.Write([]byte(tc.country))
				return
			}
			_, _ = w.Write([]byte(tc.settlement))
		}))
		c := &Client{NominatimURL: srv.URL}
		city, err := c.nominatimCity(context.Background(), tc.q)
		srv.Close()
		if tc.want == "" {
			if err != ErrNoCity {
				t.Errorf("%s: wanted no city, got %+v %v", tc.q, city, err)
			}
			continue
		}
		if err != nil || city.Name != tc.want {
			t.Errorf("%s: wanted %s, got %+v %v", tc.q, tc.want, city, err)
		}
	}
}
