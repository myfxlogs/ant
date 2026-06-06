package ai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

// AIServer implements the ant.v1.AIServiceHandler interface.
type AIServer struct {
	systemSvc     *systemai.Service
	conversations *repository.AIConversationRepository
	agentDefRepo  *repository.AIAgentDefinitionRepository
	log           *zap.Logger
}

// NewAIServer creates an AI service ConnectRPC handler.
func NewAIServer(systemSvc *systemai.Service, conversations *repository.AIConversationRepository, log *zap.Logger) *AIServer {
	return &AIServer{systemSvc: systemSvc, conversations: conversations, log: log}
}

var _ antv1c.AIServiceHandler = (*AIServer)(nil)

// userIDFromBearer extracts the user ID from an Authorization: Bearer header.
func userIDFromBearer(r *http.Request) (uuid.UUID, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return uuid.Nil, fmt.Errorf("missing bearer token")
	}
	raw := strings.TrimPrefix(auth, "Bearer ")
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return uuid.Nil, fmt.Errorf("invalid jwt format")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, fmt.Errorf("decode jwt claims: %w", err)
	}
	var claims struct {
		UserID string `json:"user_id"`
		Sub    string `json:"sub"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return uuid.Nil, fmt.Errorf("parse jwt claims: %w", err)
	}
	userID := claims.UserID
	if userID == "" {
		userID = claims.Sub
	}
	return uuid.Parse(userID)
}
