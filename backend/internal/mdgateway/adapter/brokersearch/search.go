// Package brokersearch provides a standalone mtapi Search client for broker discovery.
// Unlike the per-account Gateway types, Search does not require account credentials —
// it connects to the mtapi gRPC gateway, calls the public Search RPC, and disconnects.
package brokersearch

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	antv1 "alphaforge/gen/proto/ant/v1"
	mt4pb "alphaforge/mt4"
	mt5pb "alphaforge/mt5"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// LOWPRI-SWEEP-1 MDG-5b: mtapi default gateway literals consolidated into
// named constants (was scattered across 6 sites).
const (
	defaultMT4Gateway = "mt4grpc3.mtapi.io:443"
	defaultMT5Gateway = "mt5grpc3.mtapi.io:443"
)

// Searcher calls mtapi Search RPC for MT4 and MT5 broker discovery.
type Searcher struct {
	mt4Gateway string // e.g. "mt4grpc3.mtapi.io:443"
	mt5Gateway string // e.g. "mt5grpc3.mtapi.io:443"
}

// New creates a Searcher with explicit gateway addresses.
// If a gateway is empty, the mtapi default is used.
// Deprecated: prefer NewFromConfig for explicit config-driven construction.
// New is kept for backward compatibility.
func New(mt4Gateway, mt5Gateway string) *Searcher {
	if mt4Gateway == "" {
		mt4Gateway = defaultMT4Gateway
	}
	if mt5Gateway == "" {
		mt5Gateway = defaultMT5Gateway
	}
	return &Searcher{mt4Gateway: mt4Gateway, mt5Gateway: mt5Gateway}
}

// NewFromConfig creates a Searcher from explicit configuration values.
// BROKER-SEARCH-1 S6: unlike New, this constructor is intended for
// config/env-var-driven wiring. Empty values still fall back to the
// mtapi hardcoded defaults so existing deployments are not broken.
func NewFromConfig(mt4Gateway, mt5Gateway string) *Searcher {
	if mt4Gateway == "" {
		mt4Gateway = defaultMT4Gateway
	}
	if mt5Gateway == "" {
		mt5Gateway = defaultMT5Gateway
	}
	return &Searcher{mt4Gateway: mt4Gateway, mt5Gateway: mt5Gateway}
}

// Search returns matching broker companies from mtapi for the given company prefix.
// mtType can be "mt4", "mt5", or "" (both).
// LOWPRI-SWEEP-1 MDG-5a: fail-closed — mtapi errors propagate (stale static
// fallback removed: binding wizard must not connect to fabricated addresses),
// and an empty result is returned as-is (a legitimate "no match").
func (s *Searcher) Search(ctx context.Context, company, mtType string) ([]*antv1.BrokerCompany, error) {
	mtType = strings.ToLower(mtType)
	switch mtType {
	case "mt4":
		return s.searchMT4(ctx, company)
	case "mt5":
		return s.searchMT5(ctx, company)
	default:
		return s.searchBoth(ctx, company)
	}
}

func (s *Searcher) searchMT4(ctx context.Context, company string) ([]*antv1.BrokerCompany, error) {
	conn, err := grpc.NewClient(s.mt4Gateway,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
	)
	if err != nil {
		return nil, fmt.Errorf("brokersearch mt4 dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client := mt4pb.NewServiceClient(conn)
	reply, err := client.Search(ctx, &mt4pb.SearchRequest{Company: company})
	if err != nil {
		return nil, fmt.Errorf("brokersearch mt4 Search: %w", err)
	}
	return mapMT4Reply(reply), nil
}

func (s *Searcher) searchMT5(ctx context.Context, company string) ([]*antv1.BrokerCompany, error) {
	conn, err := grpc.NewClient(s.mt5Gateway,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
	)
	if err != nil {
		return nil, fmt.Errorf("brokersearch mt5 dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client := mt5pb.NewServiceClient(conn)
	reply, err := client.Search(ctx, &mt5pb.SearchRequest{Company: company})
	if err != nil {
		return nil, fmt.Errorf("brokersearch mt5 Search: %w", err)
	}
	return mapMT5Reply(reply), nil
}

func (s *Searcher) searchBoth(ctx context.Context, company string) ([]*antv1.BrokerCompany, error) {
	var all []*antv1.BrokerCompany
	mt4Results, err4 := s.searchMT4(ctx, company)
	all = append(all, mt4Results...)
	mt5Results, err5 := s.searchMT5(ctx, company)
	all = append(all, mt5Results...)
	if err4 != nil && err5 != nil {
		return nil, fmt.Errorf("searchBoth: mt4: %w; mt5: %w", err4, err5)
	}
	return all, nil
}

func mapMT4Reply(reply *mt4pb.SearchReply) []*antv1.BrokerCompany {
	if reply == nil {
		return nil
	}
	var companies []*antv1.BrokerCompany
	for _, c := range reply.GetResult() {
		bc := &antv1.BrokerCompany{CompanyName: c.GetCompanyName()}
		for _, r := range c.GetResults() {
			bc.Servers = append(bc.Servers, &antv1.BrokerServer{
				Name:   r.GetName(),
				Access: r.GetAccess(),
			})
		}
		companies = append(companies, bc)
	}
	return companies
}

func mapMT5Reply(reply *mt5pb.SearchReply) []*antv1.BrokerCompany {
	if reply == nil {
		return nil
	}
	var companies []*antv1.BrokerCompany
	for _, c := range reply.GetResult() {
		bc := &antv1.BrokerCompany{CompanyName: c.GetCompanyName()}
		for _, r := range c.GetResults() {
			bc.Servers = append(bc.Servers, &antv1.BrokerServer{
				Name:   r.GetName(),
				Access: r.GetAccess(),
			})
		}
		companies = append(companies, bc)
	}
	return companies
}
