package main

import (
	"context"
	"fmt"
	"time"

	"github.com/arista-northwest/go-passpersist/passpersist"
)

var (
	date    string
	tag     string
	version string
)

func runner(pp *passpersist.PassPersist) {
	var epoch time.Duration = time.Duration(time.Now().UnixNano())
	pp.AddString([]int{0}, "Hello from PassPersist")
	pp.AddString([]int{1}, "You found a secret message!")
	pp.AddTimeTicks([]int{2}, epoch)

	for i := 1; i <= 2; i++ {
		for j := 1; j <= 2; j++ {
			pp.AddString([]int{i, j}, fmt.Sprintf("Value: %d.%d", i, j))
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pp := passpersist.NewPassPersist(
		passpersist.WithRefresh(time.Second * 1),
	)

	pp.Run(ctx, runner)
}
