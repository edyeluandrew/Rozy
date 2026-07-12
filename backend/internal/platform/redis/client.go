package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *goredis.Client
}

func New(redisURL string) (*Client, error) {
	if redisURL == "" {
		return nil, fmt.Errorf("redis url empty")
	}
	opt, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	rdb := goredis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func geoKey(rideType string) string {
	return "geo:operators:" + rideType
}

func (c *Client) SetOperatorLocation(ctx context.Context, rideType, operatorID string, lng, lat float64) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.GeoAdd(ctx, geoKey(rideType), &goredis.GeoLocation{
		Name:      operatorID,
		Longitude: lng,
		Latitude:  lat,
	}).Err()
}

func (c *Client) RemoveOperator(ctx context.Context, rideType, operatorID string) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.ZRem(ctx, geoKey(rideType), operatorID).Err()
}

type NearbyOperator struct {
	OperatorID string
	DistanceKm float64
}

func (c *Client) SearchNearby(ctx context.Context, rideType string, lng, lat, radiusKm float64, limit int) ([]NearbyOperator, error) {
	res, err := c.rdb.GeoSearchLocation(ctx, geoKey(rideType), &goredis.GeoSearchLocationQuery{
		GeoSearchQuery: goredis.GeoSearchQuery{
			Longitude:  lng,
			Latitude:   lat,
			Radius:     radiusKm,
			RadiusUnit: "km",
			Sort:       "ASC",
			Count:      limit,
		},
		WithCoord: false,
		WithDist:  true,
	}).Result()
	if err != nil {
		return nil, err
	}

	out := make([]NearbyOperator, 0, len(res))
	for _, item := range res {
		out = append(out, NearbyOperator{OperatorID: item.Name, DistanceKm: item.Dist})
	}
	return out, nil
}
