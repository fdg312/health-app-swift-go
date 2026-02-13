package auth

import (
	"context"

	"github.com/fdg312/health-hub/internal/userctx"
)

func WithUserID(ctx context.Context, userID string) context.Context {
	return userctx.WithUserID(ctx, userID)
}

func GetUserID(ctx context.Context) (string, bool) {
	return userctx.GetUserID(ctx)
}

// GetOwnerUserID is kept for backward compatibility.
func GetOwnerUserID(ctx context.Context) (string, bool) {
	return GetUserID(ctx)
}
