package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/arista-northwest/go-passpersist/passpersist"
	"github.com/go-cmd/cmd"
)

func EosCommand(command string) (string, error) {
	c := cmd.NewCmd("Cli", "-p15", "-c", command)
	c.Env = append(c.Env, "TERM=dumb")
	<-c.Start()

	stderr := c.Status().Stderr
	if len(stderr) > 0 {
		return "", fmt.Errorf("%s", strings.Join(stderr, "\n"))
	}
	return strings.Join(c.Status().Stdout, "\n"), nil
}

/*
{
    "imageFormatVersion": "1.0",
    "cEosToolsVersion": "1.1",
    "uptime": 28197.233053922653,
    "modelName": "cEOSLab",
    "kernelVersion": "5.15.0-57-generic",
    "internalVersion": "4.28.1F-27567444.4281F",
    "memTotal": 65434204,
    "mfgName": "Arista",
    "serialNumber": "CB7F119E346534EA0F614DE32E9B463D",
    "systemMacAddress": "00:1c:73:2b:1f:9e",
    "bootupTimestamp": 1673459277.0984712,
    "memFree": 59489948,
    "version": "4.28.1F-27567444.4281F (engineering build)",
    "configMacAddress": "00:00:00:00:00:00",
    "isIntlVersion": false,
    "imageOptimization": "None",
    "internalBuildId": "aa54565c-ad3f-47c8-95a6-9b82f8bf7ad3",
    "hardwareRevision": "",
    "hwMacAddress": "00:00:00:00:00:00",
    "architecture": "i686"
}
*/

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

type ShowVersion struct {
	ImageFormatVersion   string       `json:"imageFormatVersion,omitempty"`
	CEOSToolsVersion     string       `json:"cEosToolsVersion,omitempty"`
	Uptime               EOSTimeTicks `json:"uptime,omitempty"`
	ModelName            string       `json:"modelName,omitempty"`
	KernelVersion        string       `json:"kernelVersion,omitempty"`
	InternalVersion      string       `json:"internalVersion,omitempty"`
	MemoryTotal          int          `json:"memTotal,omitempty"`
	Manufacturer         string       `json:"mfgName,omitempty"`
	SerialNumber         string       `json:"serialNumber,omitempty"`
	SystemMACAddress     MACAddress   `json:"systemMacAddress,omitempty"`
	BootupTimeStamp      EOSTimestamp `json:"bootupTimestamp,omitempty"`
	MemoryFree           int          `json:"memFree,omitempty"`
	Version              string       `json:"version,omitempty"`
	ConfiguredMACAddress MACAddress   `json:"configMacAddress,omitempty"`
	IsInternalVersion    bool         `json:"isIntlVersion,omitempty"`
	ImageOptimization    string       `json:"imageOptimization,omitempty"`
	InternalBuildID      string       `json:"internalBuildId,omitempty"`
	HardwareRevision     string       `json:"hardwareRevision,omitempty"`
	HardwareMACAddress   MACAddress   `json:"hwMacAddress,omitempty"`
	Architecture         string       `json:"architecture,omitempty"`
}

func main() {
	out, err := EosCommand("show version | json")
	if err != nil {
		log.Fatal(err)
	}
	text := []byte(out)
	var data ShowVersion
	json.Unmarshal(text, &data)

	passpersist.Config.Refresh = time.Second * 10
	pp := passpersist.NewPassPersist(&passpersist.Config)
	ctx := context.Background()
	pp.Run(ctx, func(pp *passpersist.PassPersist) {
		pp.AddString([]int{255, 1}, data.Version)
		pp.AddInt([]int{255, 2}, int32(data.MemoryFree))
		pp.AddCounter32([]int{255, 3}, uint32(4294967295))
		pp.AddCounter64([]int{255, 4}, uint64(18446744073709551615))
		pp.AddOID([]int{255, 5}, passpersist.OID{1, 3, 6, 1, 4, 1, 30065, 4, 224})
		pp.AddOctetString([]int{255, 6}, []byte{'0', 'b', 'c', 'd'})
		pp.AddIPAddr([]int{255, 7}, "1.2.3.4")
		pp.AddIPV6Addr([]int{255, 8}, "dead:beef:1:2:3::4")
		pp.AddGauge([]int{255, 9}, uint32(4294967295))
		pp.AddTimeTicks([]int{255, 10}, data.Uptime.Duration)
	})
}
