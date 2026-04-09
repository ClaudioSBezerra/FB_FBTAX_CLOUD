package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// ProductResponse is the public API response shape for a portal product.
type ProductResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	IconURL        string `json:"icon_url"`
	DestinationURL string `json:"destination_url"`
	Contracted     bool   `json:"contracted"`
}

// GetPortalProductsHandler returns all active products, with contracted=true
// for products belonging to the tenant identified by ?tenant=slug.
func GetPortalProductsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		tenant := r.URL.Query().Get("tenant")

		var (
			rows *sql.Rows
			err  error
		)

		if tenant == "" {
			// No tenant — all products with contracted: false
			rows, err = db.Query(`
				SELECT
					id,
					name,
					COALESCE(description, ''),
					COALESCE(icon_url, ''),
					COALESCE(destination_url, '')
				FROM portal.pt_products
				WHERE is_active = true
				ORDER BY name
			`)
		} else {
			// With tenant — LEFT JOIN to determine contracted status
			rows, err = db.Query(`
				SELECT
					p.id,
					p.name,
					COALESCE(p.description, ''),
					COALESCE(p.icon_url, ''),
					COALESCE(p.destination_url, ''),
					CASE WHEN tp.id IS NOT NULL THEN true ELSE false END AS contracted
				FROM portal.pt_products p
				LEFT JOIN portal.pt_tenant_products tp
					ON tp.product_id = p.id
					AND tp.tenant_id = (
						SELECT id FROM portal.pt_tenants WHERE slug = $1 LIMIT 1
					)
					AND tp.is_active = true
				WHERE p.is_active = true
				ORDER BY p.name
			`, tenant)
		}

		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		products := []ProductResponse{}
		for rows.Next() {
			var p ProductResponse
			var scanErr error
			if tenant == "" {
				scanErr = rows.Scan(&p.ID, &p.Name, &p.Description, &p.IconURL, &p.DestinationURL)
			} else {
				scanErr = rows.Scan(&p.ID, &p.Name, &p.Description, &p.IconURL, &p.DestinationURL, &p.Contracted)
			}
			if scanErr != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			products = append(products, p)
		}

		if err = rows.Err(); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
	}
}
