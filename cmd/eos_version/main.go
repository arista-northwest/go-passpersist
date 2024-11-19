package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"log/syslog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/arista-northwest/go-passpersist/passpersist"
	"github.com/arista-northwest/go-passpersist/utils"
	"github.com/arista-northwest/go-passpersist/utils/arista"
	"github.com/arista-northwest/go-passpersist/utils/logger"
)

var (
	date    string
	tag     string
	version string
)

type MACAddress struct {
	net.HardwareAddr
}

func (m *MACAddress) UnmarshalJSON(b []byte) error {
	str := strings.Trim(string(b), "\"")
	if str == "" {
		return nil
	}
	mac, err := net.ParseMAC(str)
	if err != nil {
		return err
	}
	*m = MACAddress{mac}
	return nil
}

type EOSTimeTicks struct {
	time.Duration
}

func (t *EOSTimeTicks) UnmarshalJSON(b []byte) error {
	ticks, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return err
	}
	tmp := time.Duration(time.Nanosecond * time.Duration(ticks*1_000_000_000))
	*t = EOSTimeTicks{tmp}
	return nil
}

type EOSTimestamp struct {
	time.Time
}

func (t *EOSTimestamp) UnmarshalJSON(b []byte) error {
	ts, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return err
	}
	tmp := time.Unix(0, int64(ts*1_000_000_000))
	*t = EOSTimestamp{tmp}
	return nil
}

type Component struct {
	Name    string
	Version string
}

type Package struct {
	Version string
	Release string
}

type Details struct {
	SystemEpoch string
	SwitchType  string
	Deviations  []string
	Components  []Component
	Packages    map[string]Package
}

type ShowVersion struct {
	Manufacturer         string       `json:"mfgName,omitempty"`
	ModelName            string       `json:",omitempty"`
	HardwareRevision     string       `json:",omitempty"`
	SerialNumber         string       `json:",omitempty"`
	SystemMACAddress     MACAddress   `json:",omitempty"`
	HardwareMACAddress   MACAddress   `json:"hwMacAddress,omitempty"`
	ConfiguredMACAddress MACAddress   `json:"configMacAddress,omitempty"`
	Version              string       `json:",omitempty"`
	Architecture         string       `json:"architecture,omitempty"`
	InternalVersion      string       `json:",omitempty"`
	InternalBuildID      string       `json:",omitempty"`
	ImageFormatVersion   string       `json:",omitempty"`
	ImageOptimization    string       `json:",omitempty"`
	BootupTimeStamp      EOSTimestamp `json:",omitempty"`
	Uptime               EOSTimeTicks `json:",omitempty"`
	MemoryTotal          uint32       `json:"memTotal,omitempty"`
	MemoryFree           uint32       `json:"memFree,omitempty"`
	IsInternalVersion    bool         `json:"isIntlVersion,omitempty"`

	Details Details

	// CEOSToolsVersion string `json:"cEosToolsVersion,omitempty"`
	// KernelVersion    string `json:"kernelVersion,omitempty"`
}

func init() {
	logger.EnableSyslogger(syslog.LOG_LOCAL4, slog.LevelInfo)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := flag.Bool("mock", false, "use mock data")
	utils.CommonCLI(version, tag, date)

	var opts []passpersist.Option
	baseOID, _ := utils.GetBaseOIDFromSNMPdConfig()
	if baseOID != nil {
		opts = append(opts, passpersist.WithBaseOID(*baseOID))
	}

	pp := passpersist.NewPassPersist(opts...)

	pp.Run(ctx, func(pp *passpersist.PassPersist) {
		var data ShowVersion

		if *mock {
			utils.MustLoadMockDataFile(&data, "mock.json")
		} else {
			err := arista.EosCommandJson("show version", data)
			if err != nil {
				slog.Error("failed to run command", slog.Any("error", err))
				os.Exit(1)
			}
		}

		pp.AddString([]int{255, 1}, data.Manufacturer)
		pp.AddString([]int{255, 2}, data.ModelName)
		pp.AddString([]int{255, 3}, data.HardwareRevision)
		pp.AddString([]int{255, 4}, data.SerialNumber)
		pp.AddString([]int{255, 5}, data.SystemMACAddress.String())
		pp.AddString([]int{255, 6}, data.HardwareMACAddress.String())
		pp.AddString([]int{255, 7}, data.ConfiguredMACAddress.String())
		pp.AddString([]int{255, 8}, data.Version)
		pp.AddString([]int{255, 9}, data.Architecture)
		pp.AddString([]int{255, 10}, data.Architecture)
		pp.AddString([]int{255, 11}, data.InternalVersion)
		pp.AddString([]int{255, 12}, data.InternalBuildID)
		pp.AddString([]int{255, 13}, data.ImageFormatVersion)
		pp.AddString([]int{255, 14}, data.ImageOptimization)
		pp.AddCounter64([]int{255, 15}, uint64(data.BootupTimeStamp.Unix()))
		pp.AddTimeTicks([]int{255, 16}, data.Uptime.Duration)
		pp.AddGauge([]int{255, 17}, data.MemoryTotal)
		pp.AddGauge([]int{255, 18}, data.MemoryTotal)
		pp.AddString([]int{255, 19}, fmt.Sprintf("%t", data.IsInternalVersion))

    pp.AddString([]int{255, 20}, data.Details.SwitchType)

    for i, c := range data.Details.Components {
        pp.AddString([]int{255, 21, 1, 1, i}, c.Name)
        pp.AddString([]int{255, 21, 1, 2, i}, c.Version)
    }

    i := 0;
    for n, p := range data.Details.Packages {
      pp.AddString([]int{255, 22, 1, 1, i}, n)
      pp.AddString([]int{255, 22, 1, 2, i}, p.Version)
      pp.AddString([]int{255, 22, 1, 2, i}, p.Release)
      i += 1
    }
	})
}
