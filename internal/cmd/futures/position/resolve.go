package position

import (
	"context"
	"fmt"
	"strconv"

	"github.com/vika2603/100x-cli/api/futures"
)

func resolvePositionID(ctx context.Context, c *futures.Client, market, positionID string) (string, error) {
	if positionID != "" {
		return positionID, nil
	}
	positions, err := c.Position.PendingPosition(ctx, futures.PendingPositionReq{Market: market})
	if err != nil {
		return "", err
	}
	if len(positions) == 0 {
		return "", fmt.Errorf("no open position found for %s", market)
	}
	if len(positions) > 1 {
		return "", fmt.Errorf("multiple open positions found for %s; pass --position-id", market)
	}
	return strconv.Itoa(positions[0].PositionID), nil
}
