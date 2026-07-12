package places

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rozy/backend/internal/platform/httpx"
)

type Place struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	LandmarkNote string  `json:"landmark_note,omitempty"`
	Category     string  `json:"category,omitempty"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Search(ctx context.Context, query string) ([]Place, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.name, COALESCE(p.landmark_note,''), COALESCE(p.category,''),
		       ST_Y(p.location::geometry), ST_X(p.location::geometry)
		FROM places p
		JOIN cities c ON c.id = p.city_id
		WHERE c.slug = 'mbarara' AND p.is_active = true
		  AND (p.name ILIKE '%' || $1 || '%' OR COALESCE(p.landmark_note,'') ILIKE '%' || $1 || '%')
		ORDER BY p.name LIMIT 20
	`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Place
	for rows.Next() {
		var p Place
		if err := rows.Scan(&p.ID, &p.Name, &p.LandmarkNote, &p.Category, &p.Lat, &p.Lng); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	places, err := h.repo.Search(r.Context(), q)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "search failed")
		return
	}
	if places == nil {
		places = []Place{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"places": places})
}
