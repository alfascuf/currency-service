package currency

import (
	"context"
	"fmt"
	"time"

	pb "github.com/alfascuf/PROD1/pkg/grpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.CurrencyServiceClient
}

func NewClient(address string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to currency service: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewCurrencyServiceClient(conn),
	}, nil
}

func (c *Client) GetRate(ctx context.Context, base, target, date string) (*pb.GetRateResponse, error) {
	return c.client.GetRate(ctx, &pb.GetRateRequest{
		Base:   base,
		Target: target,
		Date:   date,
	})
}

func (c *Client) GetHistory(ctx context.Context, base, target, startDate, endDate string) (*pb.GetHistoryResponse, error) {
	return c.client.GetHistory(ctx, &pb.GetHistoryRequest{
		Base:      base,
		Target:    target,
		StartDate: startDate,
		EndDate:   endDate,
	})
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
