package passpersist

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"

	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
}

func OIDFromString(oid string) (OID, error) {

	split := strings.Split(strings.Trim(oid, "."), ".")

	var new OID = make([]int, len(split))

	for i, p := range split {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid oid: '%s'", oid)
		}
		new[i] = n
	}
	return new, nil
}

type PassPersist struct {
	baseOid OID
	refresh time.Duration
	cache   *Cache
}

func NewPassPersist(config *ConfigT) *PassPersist {
	return &PassPersist{
		baseOid: config.BaseOid,
		refresh: config.Refresh,
		cache:   NewCache(),
	}
}

func (p *PassPersist) get(oid OID) *VarBind {
	log.Debug().Msgf("getting oid: %s", oid.String())
	return p.cache.Get(oid)
}

func (p *PassPersist) getNext(oid OID) *VarBind {
	return p.cache.GetNext(oid)
}

func (p *PassPersist) AddEntry(oid OID, value typedValue) error {
	oid = p.baseOid.Append(oid)

	log.Debug().Msgf("adding %s: %s, %s", value.TypeString(), oid, value)

	err := p.cache.Set(&VarBind{
		Oid:   oid,
		Value: value,
	})

	if err != nil {
		return err
	}

	return nil
}

func (p *PassPersist) AddString(oid OID, value string) error {
	return p.AddEntry(oid, typedValue{Value: &StringVal{Value: value}})
}

func (p *PassPersist) AddInt(oid OID, value int32) error {
	return p.AddEntry(oid, typedValue{Value: &IntVal{Value: value}})
}

func (p *PassPersist) AddOID(oid OID, value OID) error {
	return p.AddEntry(oid, typedValue{Value: &OIDVal{Value: value}})
}

func (p *PassPersist) AddOctetString(oid OID, value []byte) error {
	return p.AddEntry(oid, typedValue{Value: &OctetStringVal{Value: value}})
}

func (p *PassPersist) AddIPAddr(oid OID, value string) error {
	ip := net.ParseIP(value)
	return p.AddEntry(oid, typedValue{Value: &IPAddrVal{Value: ip}})
}

func (p *PassPersist) AddIPV6Addr(oid OID, value string) error {
	ip := net.ParseIP(value)
	return p.AddEntry(oid, typedValue{Value: &IPV6AddrVal{Value: ip}})
}

func (p *PassPersist) AddCounter32(oid OID, value uint32) error {
	return p.AddEntry(oid, typedValue{Value: &Counter32Val{Value: value}})
}

func (p *PassPersist) AddCounter64(oid OID, value uint64) error {
	return p.AddEntry(oid, typedValue{Value: &Counter64Val{Value: value}})
}

func (p *PassPersist) AddGauge(oid OID, value uint32) error {
	return p.AddEntry(oid, typedValue{Value: &GaugeVal{Value: value}})
}

func (p *PassPersist) AddTimeTicks(oid OID, value time.Duration) error {
	return p.AddEntry(oid, typedValue{Value: &TimeTicksVal{Value: value}})
}

func (p *PassPersist) Dump() {
	out := make(map[string]interface{})

	out["base-oid"] = p.baseOid
	out["refresh"] = p.refresh

	j, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(j))

	p.cache.Dump()
}
func setPrio(prio int) error {
	var err error

	switch runtime.GOOS {
	case "linux", "bsd", "freebsd", "netbsd", "openbsd":
		err = unix.Setpriority(unix.PRIO_PROCESS, 0, prio)
	}

	if err != nil {
		return err
	}

	return nil
}
func (p *PassPersist) update(ctx context.Context, callback func(*PassPersist)) {

	err := setPrio(15)
	if err != nil {
		log.Warn().Msgf("failed to set priority")
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			timer := time.NewTimer(p.refresh)

			callback(p)
			p.cache.Commit()

			<-timer.C
		}
	}
}

func (p *PassPersist) Run(ctx context.Context, f func(*PassPersist)) {
	input := make(chan string)
	done := make(chan bool)

	go p.update(ctx, f)
	go watchStdin(ctx, input, done)

	for {
		select {
		case line := <-input:
			switch line {
			case "PING":
				fmt.Println("PONG")
			case "getnext":
				inp := <-input
				if oid, ok := p.convertAndValidateOid(inp); ok {
					v := p.getNext(oid)
					if v != nil {
						fmt.Println(v.Marshal())
					} else {
						fmt.Println("NONE")
					}
				} else {
					fmt.Println("NONE")
				}

			case "get":
				inp := <-input
				if oid, ok := p.convertAndValidateOid(inp); ok {
					v := p.get(oid)
					if v != nil {
						fmt.Println(v.Marshal())
					} else {
						fmt.Println("NONE")
					}
				} else {
					fmt.Println("NONE")
				}
			case "set":
				// not-writable, wrong-type, wrong-length, wrong-value or inconsistent-value
				fmt.Println("not-writable")
			case "DUMP", "D":
				p.Dump()
			case "DUMPCACHE", "DC":
				p.cache.Dump()
			default:
				fmt.Println("NONE")
			}
		case <-done:
			return
		case <-ctx.Done():
			return
		}
	}
}

func watchStdin(ctx context.Context, input chan<- string, done chan<- bool) {

	scanner := bufio.NewScanner(os.Stdin)

	defer func() {
		done <- true
	}()

	for scanner.Scan() {

		select {
		case <-ctx.Done():
			log.Debug().Msg("ctx done")
		default:
			line := scanner.Text()
			log.Debug().Msgf("Got user input: %s", line)
			input <- line
		}
	}

	if err := scanner.Err(); err != nil {
		if err != io.EOF {
			log.Error().Msg(err.Error())
		}
	}
}

func (p *PassPersist) convertAndValidateOid(oid string) (OID, bool) {
	o, err := OIDFromString(oid)
	if err != nil {
		return o, false
	} else if !o.HasPrefix(p.baseOid) {
		return o, false
	}
	return o, true
}
