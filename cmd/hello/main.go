package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"log/syslog"
	"os"
	"runtime"
	"time"

	"github.com/arista-northwest/go-passpersist/passpersist"
	"github.com/arista-northwest/go-passpersist/utils"
	"github.com/arista-northwest/go-passpersist/utils/arista"
)

var (
	date    string
	tag     string
	version string = "dev"
)

func init() {
	w, _ := syslog.New(syslog.LOG_LOCAL4, utils.ProgName())
	l := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(l)
}

func displayVersionAndExit() {
	fmt.Printf("%s ver %s date %s tag %s [%s/%s]\n", utils.ProgName(), version, date, tag, runtime.GOOS, runtime.GOARCH)
	os.Exit(0)
}

func main() {
	var ver bool
	flag.BoolVar(&ver, "v", false, "show version")
	flag.Parse()

	if ver {
		displayVersionAndExit()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var opts []passpersist.ConfigFunc

	b, _ := arista.GetBaseOIDFromSnmpConfig()
	if b != nil {
		opts = append(opts, passpersist.WithBaseOID(*b))
	}
	opts = append(opts, passpersist.WithRefreshInterval(time.Second*300))

	pp := passpersist.NewPassPersist(ctx, opts...)

	pp.Run(func(pp *passpersist.PassPersist) {
		pp.AddString([]int{0}, "Hello from PassPersist")
		pp.AddString([]int{1}, "You found a secret message!")
		slog.Info("added strings...")

		// for i := 2; i <= 10; i++ {
		// 	for j := 1; j <= 10; j++ {
		// 		pp.AddString([]int{i, j}, fmt.Sprintf("Value: %d.%d", i, j))
		// 		slog.Debug("added string", slog.Any("subs", []int{i, j}))
		// 	}
		// }
	})
}
