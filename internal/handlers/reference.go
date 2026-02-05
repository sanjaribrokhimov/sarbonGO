// Reference (spravochniki) handlers — X-Client-Token only, no user JWT.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sarbonGO/backend/internal/response"
)

// UserCategory is a single reference item for user category.
type UserCategory struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// UserCategories returns handler for GET /reference/user-categories (list of user categories).
func UserCategories(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		rows, err := pool.Query(ctx, `
			SELECT id, code, name, sort_order
			FROM user_categories
			ORDER BY sort_order
		`)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		defer rows.Close()

		var list []UserCategory
		for rows.Next() {
			var u UserCategory
			if err := rows.Scan(&u.ID, &u.Code, &u.Name, &u.SortOrder); err != nil {
				response.Error(c, http.StatusInternalServerError, "internal error")
				return
			}
			list = append(list, u)
		}
		if err := rows.Err(); err != nil {
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		if list == nil {
			list = []UserCategory{}
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, list)
	}
}
