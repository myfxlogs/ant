package connection

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"anttrader/internal/model"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func FormatConnectionError(err error) (userMessage string, connStatus model.ConnectionStatus, detail string) {
	if err == nil {
		return "", model.ConnectionStatusSuccess, ""
	}

	detail = err.Error()
	connStatus = model.ConnectionStatusFailed

	lowDetail := strings.ToLower(detail)
	if strings.Contains(lowDetail, "id header is required") {
		return "网关鉴权失败：缺少必要的请求标识（id header）", model.ConnectionStatusFailed, detail
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "请求超时（连接/响应超时）", model.ConnectionStatusTimeout, detail
	}
	if errors.Is(err, context.Canceled) {
		return "请求已取消", model.ConnectionStatusFailed, detail
	}

	if st, ok := grpcstatus.FromError(err); ok {
		code := st.Code()
		msg := strings.TrimSpace(st.Message())
		if msg == "" {
			msg = code.String()
		}
		lowMsg := strings.ToLower(msg)
		if strings.Contains(lowMsg, "id header is required") {
			return "网关鉴权失败：缺少必要的请求标识（id header）", model.ConnectionStatusFailed, detail
		}

		switch code {
		case codes.DeadlineExceeded:
			return "请求超时（连接/响应超时）", model.ConnectionStatusTimeout, detail
		case codes.Unauthenticated:
			return "认证失败（账号/密钥无效或已过期）", model.ConnectionStatusFailed, detail
		case codes.PermissionDenied:
			return "权限不足（无权访问该资源）", model.ConnectionStatusFailed, detail
		case codes.InvalidArgument:
			if msg != "" {
				return fmt.Sprintf("参数错误：%s", msg), model.ConnectionStatusFailed, detail
			}
			return "参数错误", model.ConnectionStatusFailed, detail
		case codes.Unavailable:
			return "服务不可用（网络异常或网关不可达）", model.ConnectionStatusFailed, detail
		case codes.ResourceExhausted:
			return "请求过于频繁或资源不足（请稍后重试）", model.ConnectionStatusFailed, detail
		case codes.NotFound:
			return "目标资源不存在（账号/会话可能已失效）", model.ConnectionStatusFailed, detail
		case codes.Internal:
			return "服务内部错误（请稍后重试或联系管理员）", model.ConnectionStatusFailed, detail
		default:
			if msg != "" {
				return fmt.Sprintf("连接失败（%s）：%s", code.String(), msg), model.ConnectionStatusFailed, detail
			}
			return fmt.Sprintf("连接失败（%s）", code.String()), model.ConnectionStatusFailed, detail
		}
	}

	var nerr net.Error
	if errors.As(err, &nerr) {
		if nerr.Timeout() {
			return "网络超时（请检查网络/代理/防火墙）", model.ConnectionStatusTimeout, detail
		}
		return "网络错误（请检查网络/代理/防火墙）", model.ConnectionStatusFailed, detail
	}

	// fallback: 不直接暴露底层错误给普通用户，但 error_detail 仍会完整保留
	return "连接失败（未知原因）", model.ConnectionStatusFailed, detail
}

func (m *ConnectionManager) logConnection(userID, accountID uuid.UUID, eventType model.ConnectionEventType, status model.ConnectionStatus, message, errorDetail, serverHost string) {
	if m.logRepo == nil {
		return
	}
	if m.shouldThrottleConnectionLog(accountID, eventType, status, message, errorDetail) {
		return
	}

	host, port := m.parseHostPort(serverHost)

	log := &model.AccountConnectionLog{
		ID:          uuid.New(),
		UserID:      userID,
		AccountID:   accountID,
		EventType:   eventType,
		Status:      status,
		Message:     message,
		ErrorDetail: errorDetail,
		ServerHost:  host,
		ServerPort:  int(port),
		CreatedAt:   time.Now(),
	}

	if err := m.logRepo.CreateConnectionLog(context.Background(), log); err != nil {
		logger.Warn("Failed to log connection event", zap.Error(err))
	}
}

func (m *ConnectionManager) shouldThrottleConnectionLog(accountID uuid.UUID, eventType model.ConnectionEventType, status model.ConnectionStatus, message, errorDetail string) bool {
	if m == nil {
		return false
	}
	// Avoid alert/log storms: same account + same event signature within a short window.
	key := fmt.Sprintf("%s|%s|%s|%s|%s", accountID.String(), eventType, status, message, errorDetail)
	now := time.Now()
	const throttleWindow = 20 * time.Second

	m.connLogThrottleMu.Lock()
	defer m.connLogThrottleMu.Unlock()
	if last, ok := m.connLogThrottleLast[key]; ok && now.Sub(last) < throttleWindow {
		return true
	}
	m.connLogThrottleLast[key] = now

	// Opportunistic cleanup to keep map bounded.
	cutoff := now.Add(-2 * throttleWindow)
	for k, ts := range m.connLogThrottleLast {
		if ts.Before(cutoff) {
			delete(m.connLogThrottleLast, k)
		}
	}
	return false
}
