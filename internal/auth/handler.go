package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

// HTTPHandler holds dependencies for auth-related HTTP requests.
type HTTPHandler struct {
	authClient nexusclashv1.AuthServiceClient
}

func NewHTTPHandler(authClient nexusclashv1.AuthServiceClient) *HTTPHandler {
	return &HTTPHandler{
		authClient: authClient,
	}
}

// writeJSON is a helper function to write JSON responses, handling serialization and headers.
func (h *HTTPHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// writeError is a helper for sending structured JSON error responses.
func (h *HTTPHandler) writeError(w http.ResponseWriter, code int, message string) {
	h.writeJSON(w, code, map[string]string{"error": message})
}

// HandleRegister is the HTTP handler for the POST /register endpoint.
func (h *HTTPHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req nexusclashv1.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Make the gRPC call to the auth service.
	// We add a timeout to the context to prevent the gateway from hanging indefinitely.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.authClient.Register(ctx, &req)
	if err != nil {
		// Translate gRPC errors to HTTP status codes.
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.AlreadyExists:
			h.writeError(w, http.StatusConflict, st.Message())
		default:
			h.writeError(w, http.StatusInternalServerError, "Registration failed")
		}
		return
	}

	h.writeJSON(w, http.StatusCreated, resp)
}

// HandleLogin is the HTTP handler for the POST /login endpoint.
func (h *HTTPHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req nexusclashv1.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.authClient.Login(ctx, &req)
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			h.writeError(w, http.StatusUnauthorized, "Invalid credentials")
		default:
			h.writeError(w, http.StatusInternalServerError, "Login failed")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}
