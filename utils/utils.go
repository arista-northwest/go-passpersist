package utils

import (
	"bufio"
	"bytes"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/arista-northwest/go-passpersist/passpersist"
)

func ProgName() string {
	return filepath.Base(ProgPath())
}

func ProgPath() string {
	return os.Args[0]
}

func DisplayVersionAndExit(version string, date string, tag string) {
	if version == "" {
		version = "dev"
	}

	if date == "" {
		date = time.Unix(0, 0).Format(time.RFC3339)
	}

	if tag == "" {
		tag = "none"
	}

	fmt.Printf("%s ver %s date %s tag %s [%s/%s]\n", ProgName(), version, date, tag, runtime.GOOS, runtime.GOARCH)

	os.Exit(0)
}

func Float64ToBytes(f float64) []byte {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, f)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}

func Float64FromBytes(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)
	return float
}

func GetBaseOIDFromSNMPdConfig() (*passpersist.OID, error) {
	return getBaseOIDFromSNMPdConfig("")
}

// path option is only used for the test case
func getBaseOIDFromSNMPdConfig(path string) (*passpersist.OID, error) {
	t := []string{"-Dread_config", "-H"}
	if path != "" {
		t = append(t, "-c", path)
	}
	cmd := exec.Command("snmpd", t...)
	p, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`pass_persist ([\d\.]+) ` + regexp.QuoteMeta(ProgPath()))

	s := bufio.NewScanner(p)
	for s.Scan() {
		b := s.Bytes()
		if re.Match(b) {
			g := re.FindSubmatch(b)
			slog.Debug("matched pass_persist in snmpd config:", "line", string(b))
			o, err := passpersist.NewOID(string(g[1]))
			return &o, err
		}
	}

	return nil, errors.New("failed to find extension in snmpd config")
}

func EncodeString(s string) []int {
	b, _ := asn1.Marshal(s)
	oid := make([]int, len(b))
	for i, b := range b {
		oid[i] = int(b)
	}
	return oid
}

// func DecodeOID(oid passpersist.OID) string {
// 	asn1.Unmarshal()
// }

func CapPanic() {
	if r := recover(); r != nil {
		slog.Error("panicing", "error", string(debug.Stack()))
		os.Exit(1)
	}
}
