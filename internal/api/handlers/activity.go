package handlers

import (
	"net/http"
	"strconv"

	"github.com/gforce/gforce/internal/api/middleware"
	"github.com/gforce/gforce/internal/api/response"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"go.uber.org/zap"
)

// ActivityHandler serves the user activity feed.
type ActivityHandler struct {
	store  store.Store
	logger *zap.Logger
}

// NewActivityHandler creates an ActivityHandler.
func NewActivityHandler(s store.Store, logger *zap.Logger) *ActivityHandler {
	return &ActivityHandler{store: s, logger: logger}
}

// ListMyActivity handles GET /api/v1/user/activity.
func (h *ActivityHandler) ListMyActivity(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	events, err := h.store.ListUserActivity(r.Context(), user.ID, limit)
	if err != nil {
		response.InternalError(w, h.logger, err)
		return
	}

	if events == nil {
		events = []*models.ActivityEvent{}
	}

	response.JSON(w, http.StatusOK, events)
}
