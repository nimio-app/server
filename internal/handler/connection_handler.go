package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nimio/server/internal/domain"
	"github.com/nimio/server/internal/middleware"
	"github.com/nimio/server/internal/service"
)

// ConnectionHandler handles connection-related endpoints
type ConnectionHandler struct {
	connectionService service.ConnectionService
}

// NewConnectionHandler creates a new connection handler
func NewConnectionHandler(connectionService service.ConnectionService) *ConnectionHandler {
	return &ConnectionHandler{
		connectionService: connectionService,
	}
}

// SendFriendRequestRequest represents the friend request body
type SendFriendRequestRequest struct {
	ToUserID         string `json:"to_user_id"`
	RelationshipTier string `json:"relationship_tier,omitempty"`
}

// AcceptFriendRequestRequest represents the accept request body
type AcceptFriendRequestRequest struct {
	FromUserID string `json:"from_user_id"`
}

// UpdateTierRequest represents the update tier request body
type UpdateTierRequest struct {
	FriendID         string `json:"friend_id"`
	RelationshipTier string `json:"relationship_tier"`
}

// SendFriendRequest handles POST /v1/connections/request
func (h *ConnectionHandler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SendFriendRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	if req.ToUserID == "" {
		ValidationErrorResponse(w, "to_user_id is required")
		return
	}

	toUserID, err := uuid.Parse(req.ToUserID)
	if err != nil {
		ValidationErrorResponse(w, "invalid to_user_id format")
		return
	}

	// Default to MUTUAL if not specified
	tier := domain.RelationshipMutual
	if req.RelationshipTier != "" {
		tier = domain.RelationshipTier(req.RelationshipTier)
		// Validate tier
		if tier != domain.RelationshipAll && tier != domain.RelationshipCircle && tier != domain.RelationshipMutual {
			ValidationErrorResponse(w, "invalid relationship_tier (must be ALL, CIRCLE, or MUTUAL)")
			return
		}
	}

	connection, err := h.connectionService.SendFriendRequest(r.Context(), userID, toUserID, tier)
	if err != nil {
		if err == domain.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "user not found")
		} else if err == domain.ErrAlreadyExists {
			ErrorResponse(w, http.StatusConflict, "friend request already exists")
		} else if err == domain.ErrForbidden {
			ErrorResponse(w, http.StatusForbidden, "connection blocked")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to send friend request")
		}
		return
	}

	SuccessResponse(w, http.StatusCreated, map[string]interface{}{
		"connection": connection,
		"message":    "Friend request sent successfully",
	})
}

// AcceptFriendRequest handles POST /v1/connections/accept
func (h *ConnectionHandler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req AcceptFriendRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	if req.FromUserID == "" {
		ValidationErrorResponse(w, "from_user_id is required")
		return
	}

	fromUserID, err := uuid.Parse(req.FromUserID)
	if err != nil {
		ValidationErrorResponse(w, "invalid from_user_id format")
		return
	}

	connection, err := h.connectionService.AcceptFriendRequest(r.Context(), userID, fromUserID)
	if err != nil {
		if err == domain.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "friend request not found")
		} else if err == domain.ErrForbidden {
			ErrorResponse(w, http.StatusForbidden, "cannot accept this request")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to accept friend request")
		}
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]interface{}{
		"connection": connection,
		"message":    "Friend request accepted successfully",
	})
}

// RejectFriendRequest handles POST /v1/connections/reject
func (h *ConnectionHandler) RejectFriendRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req AcceptFriendRequestRequest // Reuse same structure
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	if req.FromUserID == "" {
		ValidationErrorResponse(w, "from_user_id is required")
		return
	}

	fromUserID, err := uuid.Parse(req.FromUserID)
	if err != nil {
		ValidationErrorResponse(w, "invalid from_user_id format")
		return
	}

	err = h.connectionService.RejectFriendRequest(r.Context(), userID, fromUserID)
	if err != nil {
		if err == domain.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "friend request not found")
		} else if err == domain.ErrForbidden {
			ErrorResponse(w, http.StatusForbidden, "cannot reject this request")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to reject friend request")
		}
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]string{
		"message": "Friend request rejected successfully",
	})
}

// BlockUser handles POST /v1/connections/block
func (h *ConnectionHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	if req.UserID == "" {
		ValidationErrorResponse(w, "user_id is required")
		return
	}

	blockedUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		ValidationErrorResponse(w, "invalid user_id format")
		return
	}

	connection, err := h.connectionService.BlockUser(r.Context(), userID, blockedUserID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to block user")
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]interface{}{
		"connection": connection,
		"message":    "User blocked successfully",
	})
}

// RemoveConnection handles DELETE /v1/connections/:friendId
func (h *ConnectionHandler) RemoveConnection(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	friendIDStr := chi.URLParam(r, "friendId")
	if friendIDStr == "" {
		ValidationErrorResponse(w, "friend_id is required")
		return
	}

	friendID, err := uuid.Parse(friendIDStr)
	if err != nil {
		ValidationErrorResponse(w, "invalid friend_id format")
		return
	}

	err = h.connectionService.RemoveConnection(r.Context(), userID, friendID)
	if err != nil {
		if err == domain.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "connection not found")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to remove connection")
		}
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]string{
		"message": "Connection removed successfully",
	})
}

// UpdateRelationshipTier handles PUT /v1/connections/tier
func (h *ConnectionHandler) UpdateRelationshipTier(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateTierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ValidationErrorResponse(w, "invalid request body")
		return
	}

	if req.FriendID == "" {
		ValidationErrorResponse(w, "friend_id is required")
		return
	}

	if req.RelationshipTier == "" {
		ValidationErrorResponse(w, "relationship_tier is required")
		return
	}

	friendID, err := uuid.Parse(req.FriendID)
	if err != nil {
		ValidationErrorResponse(w, "invalid friend_id format")
		return
	}

	tier := domain.RelationshipTier(req.RelationshipTier)
	if tier != domain.RelationshipAll && tier != domain.RelationshipCircle && tier != domain.RelationshipMutual {
		ValidationErrorResponse(w, "invalid relationship_tier (must be ALL, CIRCLE, or MUTUAL)")
		return
	}

	connection, err := h.connectionService.UpdateRelationshipTier(r.Context(), userID, friendID, tier)
	if err != nil {
		if err == domain.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "connection not found")
		} else if err == domain.ErrForbidden {
			ErrorResponse(w, http.StatusForbidden, "not authorized to update this connection")
		} else {
			ErrorResponse(w, http.StatusInternalServerError, "failed to update relationship tier")
		}
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]interface{}{
		"connection": connection,
		"message":    "Relationship tier updated successfully",
	})
}

// GetMyConnections handles GET /v1/connections
func (h *ConnectionHandler) GetMyConnections(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Optional status filter from query params
	statusParam := r.URL.Query().Get("status")
	var status domain.ConnectionStatus
	if statusParam != "" {
		status = domain.ConnectionStatus(statusParam)
		// Validate status
		if status != domain.ConnectionPending && status != domain.ConnectionAccepted && status != domain.ConnectionBlocked {
			ValidationErrorResponse(w, "invalid status (must be PENDING, ACCEPTED, or BLOCKED)")
			return
		}
	}

	connections, err := h.connectionService.GetMyConnections(r.Context(), userID, status)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to get connections")
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]interface{}{
		"connections": connections,
		"count":       len(connections),
	})
}

// GetConnectionStatus handles GET /v1/connections/status/:userId
func (h *ConnectionHandler) GetConnectionStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	otherUserIDStr := chi.URLParam(r, "userId")
	if otherUserIDStr == "" {
		ValidationErrorResponse(w, "user_id is required")
		return
	}

	otherUserID, err := uuid.Parse(otherUserIDStr)
	if err != nil {
		ValidationErrorResponse(w, "invalid user_id format")
		return
	}

	connection, err := h.connectionService.GetConnectionStatus(r.Context(), userID, otherUserID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to get connection status")
		return
	}

	if connection == nil {
		SuccessResponse(w, http.StatusOK, map[string]interface{}{
			"connection": nil,
			"status":     "none",
		})
		return
	}

	SuccessResponse(w, http.StatusOK, map[string]interface{}{
		"connection": connection,
		"status":     connection.Status,
	})
}
