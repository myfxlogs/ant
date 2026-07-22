package mt4

import (
	"context"
	"fmt"

	pb "alphaforge/mt4"
	"google.golang.org/grpc/metadata"
)

// TickValueWithSize returns the tick value and size for the requested symbols.
func (g *Gateway) TickValueWithSize(ctx context.Context, symbols []string) ([]*pb.TickValueWithSize, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4: not connected")
	}

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	resp, err := client.TickValueWithSize(metadata.NewOutgoingContext(ctx, md), &pb.TickValueWithSizeRequest{
		Id:     sid,
		Symbol: symbols,
	})
	if err != nil {
		return nil, fmt.Errorf("mt4 TickValueWithSize: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt4 TickValueWithSize: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}
