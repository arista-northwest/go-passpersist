# go-passpersist

Golang implementation of SNMP's Pass-Persist protocol

### Example

```
package main

import (
	"context"
	"fmt"
	"log/slog"
	"log/syslog"
	"time"

	"github.com/arista-northwest/go-passpersist/passpersist"
	"github.com/arista-northwest/go-passpersist/utils"
	"github.com/arista-northwest/go-passpersist/utils/logger"
)

/*
do not edit, initialized at build time

e.g.

go build \
	-ldflags "-X main.tag=$(BUILD_TAG) -X main.date=$(BUILD_DATE) -X main.version=$(RELEASE_VER)" \
	-o $(DIST)/$(path)/$(fullname) .
*/
var (
	date    string
	tag     string
	version string
)

// Enable syslogging as early as possible
func init() {
	logger.EnableSyslogger(syslog.LOG_LOCAL4, slog.LevelInfo)
}

func main() {
	// log panics
	defer utils.CapPanic()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// optionally enable CLI flags
	utils.CommonCLI(version, tag, date)

	var opts []passpersist.Option

	// try to find the OID defined in the snmpd config, otherwise use env or default
	// override default with `PASSPERSIST_BASE_OID`. 
	// 
	// example:
	//   PASSPERSIST_BASE_OID=1.3.6.1.4.1.8072.1.3.1.226 go run .
	b, _ := utils.GetBaseOIDFromSNMPdConfig()
	if b != nil {
		opts = append(opts, passpersist.WithBaseOID(*b))
	}

	// override with `PASSPERSIST_REFRESH_RATE`
	// 
	// example:
	//    PASSPERSIST_REFRESH_RATE=1m go run .
	opts = append(opts, passpersist.WithRefresh(time.Second*300))

	pp := passpersist.NewPassPersist(opts...)

	pp.Run(ctx, func(pp *passpersist.PassPersist) {
		slog.Debug("update triggered.")
		pp.AddString([]int{0}, "Hello from PassPersist")
		pp.AddString([]int{1}, "You found a secret message!")

		for i := 1; i <= 2; i++ {
			for j := 1; j <= 2; j++ {
				pp.AddString([]int{i, j}, fmt.Sprintf("Value: %d.%d", i, j))
				slog.Debug("added string", slog.Any("subs", []int{i, j}))
			}
		}
	})
}

```
