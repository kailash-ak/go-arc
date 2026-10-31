//ticker is like an alarm that rings repeatedly after a fixed  time interval

package main

import (
	"context"
	"fmt"
	"time"
)

func worker(ctx context.Context) {
	ticker := time.NewTicker(800 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("stopped", ctx.Err())
			return

		case <-ticker.C:
			fmt.Println("working")
		}
	}
}
