package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourname/gatelink-engine/internal/db"
)

// EventsHandler 账号事件查询
type EventsHandler struct {
	db *db.Pool
}

func NewEventsHandler(db *db.Pool) *EventsHandler {
	return &EventsHandler{db: db}
}

// HandleAccountEvents GET /internal/v1/accounts/:id/events
// 返回账号最近的所有事件（健康 + 对账）
func (h *EventsHandler) HandleAccountEvents(c *gin.Context) {
	accountID := c.Param("id")

	limitStr := c.DefaultQuery("limit", "50")
	limit := 50
	if n, err := parseInt(limitStr); err == nil && n > 0 && n <= 200 {
		limit = n
	}

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT event_type, score_delta, score_after, detail, created_at
		FROM health_events
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, accountID, limit)
	if err != nil {
		InternalError(c)
		return
	}
	defer rows.Close()

	type event struct {
		Type       string    `json:"type"`
		Delta      int       `json:"delta"`
		ScoreAfter int       `json:"score_after"`
		Detail     *string   `json:"detail,omitempty"`
		CreatedAt  time.Time `json:"created_at"`
	}
	var events []event
	for rows.Next() {
		var e event
		if err := rows.Scan(&e.Type, &e.Delta, &e.ScoreAfter, &e.Detail, &e.CreatedAt); err != nil {
			continue
		}
		events = append(events, e)
	}

	OK(c, gin.H{
		"account_id": accountID,
		"events":     events,
		"count":      len(events),
	})
}

func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errInvalidInt
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

var errInvalidInt = &parseError{"invalid integer"}

type parseError struct{ msg string }

func (e *parseError) Error() string { return e.msg }
