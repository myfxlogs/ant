package mt5client

import (
	"context"
	"fmt"

	pb "anttrader/mt5"
)

func (c *MT5Client) Search(ctx context.Context, company string) ([]BrokerCompany, error) {
	grpcConn, err := c.getSearchConn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MT5 gateway: %w", err)
	}

	serviceClient := pb.NewServiceClient(grpcConn)
	resp, err := serviceClient.Search(ctx, &pb.SearchRequest{
		Company: company,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if resp.GetError() != nil {
		return nil, fmt.Errorf("MT5 error: %s", resp.GetError().GetMessage())
	}

	var companies []BrokerCompany
	for _, c := range resp.GetResult() {
		var results []BrokerResult
		for _, r := range c.GetResults() {
			results = append(results, BrokerResult{
				Name:   r.GetName(),
				Access: r.GetAccess(),
			})
		}
		companies = append(companies, BrokerCompany{
			CompanyName: c.GetCompanyName(),
			Results:     results,
		})
	}

	return companies, nil
}
