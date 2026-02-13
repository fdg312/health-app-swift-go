package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/fdg312/health-hub/internal/auth"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

var (
	ErrProfileNotOwned = errors.New("profile not owned by current user")
	ErrProfileNotFound = errors.New("profile not found")
)

// requireProfileOwned проверяет, что профиль принадлежит текущему пользователю
// Если AUTH_ENABLED=0, проверка пропускается
func requireProfileOwned(ctx context.Context, storage storage.Storage, profileID uuid.UUID, authEnabled bool) error {
	if !authEnabled {
		// Auth disabled - skip ownership check
		return nil
	}

	// Get owner_user_id from context
	ownerUserID, ok := auth.GetOwnerUserID(ctx)
	if !ok {
		// No auth context - should not happen if middleware is working
		return ErrProfileNotOwned
	}

	// Get profile and check ownership
	profile, err := storage.GetProfile(ctx, profileID)
	if err != nil {
		return ErrProfileNotFound
	}

	// Check if profile belongs to this owner_user_id
	if profile.OwnerUserID == "" || profile.OwnerUserID != ownerUserID {
		return ErrProfileNotOwned
	}

	return nil
}

// writeOwnershipError writes a 404 response for ownership violations
// (using 404 instead of 403 for security reasons - don't reveal profile existence)
func writeOwnershipError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error":{"code":"profile_not_found","message":"Profile not found"}}`))
}
