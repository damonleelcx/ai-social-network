package httpapi

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// The map on the app's first screen is public, and this is a dating app, so it only ever
// shows crowds: areas about 500 km across, none with fewer than mapMinPeople, and counts
// rounded down so that one person joining or leaving doesn't show.
const (
	mapCellDegrees = 5
	mapMinPeople   = 5
	mapCacheFor    = 10 * time.Minute
)

type mapCell struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	N   int     `json:"n"`
}

type userMap struct {
	Total int       `json:"total"`
	Cells []mapCell `json:"cells"`
}

type userMapCache struct {
	mu   sync.Mutex
	at   time.Time
	data userMap
}

func (s *Server) loadUserMap(ctx context.Context) (userMap, error) {
	out := userMap{Cells: []mapCell{}}
	// Paused and held profiles have stepped back, so they aren't counted in an area.
	rows, err := s.Pool.Query(ctx, `
		SELECT floor(p.city_lat / $1)::int, floor(p.city_lon / $1)::int, count(*)
		FROM profiles p JOIN users u ON u.id = p.user_id
		WHERE u.deleted_at IS NULL AND u.email_verified_at IS NOT NULL
		  AND p.city_lat IS NOT NULL AND p.city_lon IS NOT NULL AND p.status NOT IN ('paused', 'held')
		GROUP BY 1, 2 HAVING count(*) >= $2 ORDER BY 1, 2`, mapCellDegrees, mapMinPeople)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var lat, lon, n int
		if err := rows.Scan(&lat, &lon, &n); err != nil {
			return out, err
		}
		out.Cells = append(out.Cells, mapCell{
			Lat: float64(lat*mapCellDegrees) + mapCellDegrees/2.0,
			Lon: float64(lon*mapCellDegrees) + mapCellDegrees/2.0,
			N:   n / mapMinPeople * mapMinPeople,
		})
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE deleted_at IS NULL AND email_verified_at IS NOT NULL`).Scan(&out.Total)
	return out, err
}

func (s *Server) userMap(w http.ResponseWriter, r *http.Request) {
	c := &s.mapCache
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.at.IsZero() || time.Since(c.at) > mapCacheFor {
		data, err := s.loadUserMap(r.Context())
		if err != nil {
			handleErr(w, r, err)
			return
		}
		c.data, c.at = data, time.Now()
	}
	writeJSON(w, 200, c.data)
}
