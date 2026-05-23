package service

import (
	"context"
	"errors"
	"sort"
	"sync"

	"anttrader/internal/config"
	"anttrader/internal/mt4client"
	"anttrader/internal/mt5client"
)

var (
	ErrBrokerSearchFailed = errors.New("broker search failed")
)

type BrokerService struct {
	mt4Config *config.MT4Config
	mt5Config *config.MT5Config
	
	mt4Client *mt4client.MT4Client
	mt5Client *mt5client.MT5Client
	
	mt4ConnOnce sync.Once
	mt5ConnOnce sync.Once
}

func NewBrokerService(mt4Config *config.MT4Config, mt5Config *config.MT5Config) *BrokerService {
	return &BrokerService{
		mt4Config: mt4Config,
		mt5Config: mt5Config,
	}
}

func (s *BrokerService) getMT5Client() *mt5client.MT5Client {
	s.mt5ConnOnce.Do(func() {
		s.mt5Client = mt5client.NewMT5Client(s.mt5Config)
	})
	return s.mt5Client
}

func (s *BrokerService) getMT4Client() *mt4client.MT4Client {
	s.mt4ConnOnce.Do(func() {
		s.mt4Client = mt4client.NewMT4Client(s.mt4Config)
	})
	return s.mt4Client
}

type BrokerCompany struct {
	CompanyName string
	Results     []BrokerResult
}

type BrokerResult struct {
	Name   string
	Access []string
}

func (s *BrokerService) Search(ctx context.Context, platform, company string) ([]BrokerCompany, error) {
	if platform == "MT5" {
		return s.searchMT5(ctx, company)
	} else if platform == "MT4" {
		return s.searchMT4(ctx, company)
	}
	return nil, ErrBrokerSearchFailed
}

func (s *BrokerService) searchMT5(ctx context.Context, company string) ([]BrokerCompany, error) {
	client := s.getMT5Client()
	result, err := client.Search(ctx, company)
	if err != nil {
		return nil, err
	}

	var companies []BrokerCompany
	for _, c := range result {
		companies = append(companies, BrokerCompany{
			CompanyName: c.CompanyName,
			Results:     convertResults(c.Results),
		})
	}
	return companies, nil
}

func (s *BrokerService) searchMT4(ctx context.Context, company string) ([]BrokerCompany, error) {
	client := s.getMT4Client()
	result, err := client.Search(ctx, company)
	if err != nil {
		return nil, err
	}

	var companies []BrokerCompany
	for _, c := range result {
		companies = append(companies, BrokerCompany{
			CompanyName: c.CompanyName,
			Results:     convertResultsMT4(c.Results),
		})
	}
	return companies, nil
}

func convertResults(results []mt5client.BrokerResult) []BrokerResult {
	var out []BrokerResult
	for _, r := range results {
		out = append(out, BrokerResult{
			Name:   r.Name,
			Access: r.Access,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func convertResultsMT4(results []mt4client.BrokerResult) []BrokerResult {
	var out []BrokerResult
	for _, r := range results {
		var access []string
		for _, a := range r.Access {
			access = append(access, a+":443")
		}
		out = append(out, BrokerResult{
			Name:   r.Name,
			Access: access,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
