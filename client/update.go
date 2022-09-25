package client

import (
	"context"
	"time"
)

type UpdateInfo struct {
	Current int
	Total   int
}

func (u *UpdateInfo) GetRatio() float64 {
	return float64(u.Current) / float64(u.Total)
}

var UpdateInf *UpdateInfo

func init() {
	UpdateInf = &UpdateInfo{}
}

func update(ctx context.Context) {
	UpdateInf.Total = 100
	UpdateInf.Current = 0

	for {
		select {
		case <-time.After(100 * time.Millisecond):
			if UpdateInf.Current < UpdateInf.Total {
				UpdateInf.Current += 5
			} else {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
