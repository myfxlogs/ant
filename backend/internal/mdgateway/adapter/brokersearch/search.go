// Package brokersearch provides a standalone mtapi Search client for broker discovery.
// Unlike the per-account Gateway types, Search does not require account credentials —
// it connects to the mtapi gRPC gateway, calls the public Search RPC, and disconnects.
package brokersearch

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	mt4pb "anttrader/mt4"
	mt5pb "anttrader/mt5"
	antv1 "anttrader/gen/proto/ant/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// staticBrokers is a fallback list of well-known brokers used when the mtapi
// Search RPC is unavailable or returns no results.
var staticBrokers = []*antv1.BrokerCompany{
	{CompanyName: "Exness Technologies Ltd", Servers: []*antv1.BrokerServer{
		{Name: "Exness-Real", Access: []string{"18.163.85.196:443"}},
		{Name: "Exness-Demo", Access: []string{"18.163.85.196:443"}},
	}},
	{CompanyName: "IC Markets", Servers: []*antv1.BrokerServer{
		{Name: "ICMarkets-Live", Access: []string{}},
		{Name: "ICMarkets-Demo", Access: []string{}},
	}},
	{CompanyName: "Pepperstone", Servers: []*antv1.BrokerServer{
		{Name: "Pepperstone-Live", Access: []string{}},
		{Name: "Pepperstone-Demo", Access: []string{}},
	}},
	{CompanyName: "XM", Servers: []*antv1.BrokerServer{
		{Name: "XM-Real", Access: []string{}},
		{Name: "XM-Demo", Access: []string{}},
	}},
	{CompanyName: "RoboForex", Servers: []*antv1.BrokerServer{
		{Name: "RoboForex-Pro", Access: []string{}},
		{Name: "RoboForex-Demo", Access: []string{}},
	}},
}

// Searcher calls mtapi Search RPC for MT4 and MT5 broker discovery.
type Searcher struct {
	mt4Gateway string // e.g. "mt4grpc3.mtapi.io:443"
	mt5Gateway string // e.g. "mt5grpc3.mtapi.io:443"
}

// New creates a Searcher with explicit gateway addresses.
// If a gateway is empty, the mtapi default is used.
func New(mt4Gateway, mt5Gateway string) *Searcher {
	if mt4Gateway == "" {
		mt4Gateway = "mt4grpc3.mtapi.io:443"
	}
	if mt5Gateway == "" {
		mt5Gateway = "mt5grpc3.mtapi.io:443"
	}
	return &Searcher{mt4Gateway: mt4Gateway, mt5Gateway: mt5Gateway}
}

// Search returns matching broker companies from mtapi for the given company prefix.
// mtType can be "mt4", "mt5", or "" (both).
// When mtapi is unavailable or returns no results, it falls back to a static list of
// well-known brokers.
func (s *Searcher) Search(ctx context.Context, company, mtType string) ([]*antv1.BrokerCompany, error) {
	mtType = strings.ToLower(mtType)
	var results []*antv1.BrokerCompany
	var err error
	switch mtType {
	case "mt4":
		results, err = s.searchMT4(ctx, company)
	case "mt5":
		results, err = s.searchMT5(ctx, company)
	default:
		results, err = s.searchBoth(ctx, company)
	}
	if len(results) == 0 {
		if err != nil {
			zap.L().Warn("broker search: mtapi error, falling back to static list",
				zap.String("company", company), zap.String("mtType", mtType), zap.Error(err))
		} else {
			zap.L().Debug("broker search: no results, falling back to static list",
				zap.String("company", company), zap.String("mtType", mtType))
		}
		return staticBrokerFilter(company), nil
	}
	return results, err
}

func (s *Searcher) searchMT4(ctx context.Context, company string) ([]*antv1.BrokerCompany, error) {
	conn, err := grpc.NewClient(s.mt4Gateway,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
	)
	if err != nil {
		return nil, fmt.Errorf("brokersearch mt4 dial: %w", err)
	}
	defer conn.Close()

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
	defer conn.Close()

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

// staticBrokerFilter returns entries from the static broker list whose
// CompanyName contains the given prefix (case-insensitive). An empty
// prefix matches all entries.
func staticBrokerFilter(prefix string) []*antv1.BrokerCompany {
	if prefix == "" {
		return staticBrokers
	}
	lower := strings.ToLower(prefix)
	var out []*antv1.BrokerCompany
	for _, bc := range staticBrokers {
		if strings.Contains(strings.ToLower(bc.CompanyName), lower) {
			out = append(out, bc)
		}
	}
	return out
}
