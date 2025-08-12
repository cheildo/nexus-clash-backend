package playerprofile

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

// HTTPHandler holds dependencies for profile-related HTTP requests.
type HTTPHandler struct {
	profileClient nexusclashv1.PlayerProfileServiceClient
}

func NewHTTPHandler(profileClient nexusclashv1.PlayerProfileServiceClient) *HTTPHandler {
	return &HTTPHandler{
		profileClient: profileClient,
	}
}

// writeJSON and writeError helpers can be refactored into a shared package later
// to avoid duplication, but for now we'll keep them here for clarity.
func (h *HTTPHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, code int, message string) {
	h.writeJSON(w, code, map[string]string{"error": message})
}

// HandleGetProfile is the HTTP handler for GET /profiles/{userID}.
// It retrieves a player's profile information.
func (h *HTTPHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	// Chi's URLParam function allows us to easily extract the user ID from the URL.
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		h.writeError(w, http.StatusBadRequest, "User ID is required in the URL path")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	req := &nexusclashv1.GetProfileRequest{
		UserId: &nexusclashv1.UUID{Value: userID},
	}

	// Make the gRPC call to the player-profile-service.
	resp, err := h.profileClient.GetProfile(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			h.writeError(w, http.StatusNotFound, st.Message())
		default:
			h.writeError(w, http.StatusInternalServerError, "Failed to retrieve profile")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, resp.GetProfile())
}
