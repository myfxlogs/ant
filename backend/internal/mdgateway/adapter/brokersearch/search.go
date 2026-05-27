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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

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
	conn, err := grpc.DialContext(ctx, s.mt4Gateway,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithBlock(),
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
	conn, err := grpc.DialContext(ctx, s.mt5Gateway,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithBlock(),
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
	mt4Results, _ := s.searchMT4(ctx, company)
	all = append(all, mt4Results...)
	mt5Results, _ := s.searchMT5(ctx, company)
	all = append(all, mt5Results...)
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
