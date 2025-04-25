package passpersist

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"os"
	"time"
)

type SetError int

const (
	NotWriteable SetError = iota
	// WrongType
	// WrongValue
	// WrongLength
	// InconsistentValue
)

const (
	NetSnmpExtendMib    = "1.3.6.1.4.1.8072.1.3.1"
	NetSnmpPassExamples = "1.3.6.1.4.1.8072.2.255"
)

var (
	DefaultBaseOID     OID           = MustNewOID(NetSnmpExtendMib).MustAppend([]int{226})
	DefaultRefreshRate time.Duration = time.Second * 60
)

func init() {
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     slog.LevelError,
		AddSource: false,
	}))
	slog.SetDefault(l)
}

func (e SetError) String() string {
	switch e {
	case NotWriteable:
		return "not-writable"
	// case WrongType:
	// 	return "wrong-type"
	// case WrongValue:
	// 	return "wrong-value"
	// case WrongLength:
	// 	return "wrong-length"
	// case InconsistentValue:
	// 	return "inconsistent-value"
	default:
		slog.Warn("unknown value type id", slog.Any("error", e))
	}
	return "unknown-error"
}

type Option func(*PassPersist)

func WithRefresh(d time.Duration) func(*PassPersist) {
	return func(p *PassPersist) {
		p.refreshRate = d
	}
}

func WithBaseOID(o OID) func(*PassPersist) {
	return func(p *PassPersist) {
		p.baseOID = o
	}
}

type PassPersist struct {
	cache       *Cache
	baseOID     OID
	refreshRate time.Duration
}

func NewPassPersist(opts ...Option) *PassPersist {

	p := &PassPersist{
		cache:       NewCache(),
		baseOID:     DefaultBaseOID,
		refreshRate: DefaultRefreshRate,
	}

	for _, fn := range opts {
		fn(p)
	}

	p.overrideFromEnv()

	return p
}

func (p *PassPersist) AddEntry(subs []int, value typedValue) error {
	oid, err := p.baseOID.Append(subs)
	if err != nil {
		return err
	}

	slog.Debug("adding entry", slog.Any("value", value))

	err = p.cache.Set(&VarBind{
		OID:       oid,
		ValueType: value.TypeString(),
		Value:     value,
	})

	if err != nil {
		return err
	}

	return nil
}

func (p *PassPersist) AddString(subIds []int, value string) error {
	return p.AddEntry(subIds, typedValue{&StringVal{value}})
}

func (p *PassPersist) AddInt(subIds []int, value int32) error {
	return p.AddEntry(subIds, typedValue{&IntVal{value}})
}

func (p *PassPersist) AddOID(subIds []int, value OID) error {
	return p.AddEntry(subIds, typedValue{&OIDVal{value}})
}

func (p *PassPersist) AddOctetString(subIds []int, value []byte) error {
	return p.AddEntry(subIds, typedValue{&OctetStringVal{value}})
}

func (p *PassPersist) AddIP(subIds []int, value netip.Addr) error {
	return p.AddEntry(subIds, typedValue{&IPAddrVal{value}})
}

func (p *PassPersist) AddIPV6(subIds []int, value netip.Addr) error {
	return p.AddEntry(subIds, typedValue{&IPV6AddrVal{value}})
}

func (p *PassPersist) AddCounter32(subIds []int, value uint32) error {
	return p.AddEntry(subIds, typedValue{&Counter32Val{value}})
}

func (p *PassPersist) AddCounter64(subIds []int, value uint64) error {
	return p.AddEntry(subIds, typedValue{&Counter64Val{value}})
}

func (p *PassPersist) AddGauge(subIds []int, value uint32) error {
	return p.AddEntry(subIds, typedValue{&GaugeVal{value}})
}

func (p *PassPersist) AddTimeTicks(subIds []int, value time.Duration) error {
	return p.AddEntry(subIds, typedValue{&TimeTicksVal{value}})
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
				slog.Debug("validating", "input", inp)
				oid, err := convertAndValidateOID(inp, p.baseOID)
				if err != nil {
					slog.Warn("failed to validate input", "input", slog.Any("error", err))
					fmt.Println("NONE")
				} else {
					slog.Debug("getNext", "oid", oid.String())
					v := p.getNext(oid)
					if v != nil {
						fmt.Println(v.Marshal())
					} else {
						fmt.Println("NONE")
					}
				}

			case "get":
				inp := <-input
				oid, err := convertAndValidateOID(inp, p.baseOID)
				if err != nil {
					slog.Warn("failed to validate input", "input", slog.Any("error", err))
					fmt.Println("NONE")
				} else {
					slog.Debug("get", "oid", oid.String())
					v := p.get(oid)
					if v != nil {
						fmt.Println(v.Marshal())
					} else {
						fmt.Println("NONE")
					}
				}
			case "set":
				fmt.Println(NotWriteable.String())
			case "DUMP", "C":
				p.cache.Dump()
			case "DUMPINDEX", "I":
				p.cache.DumpIndex()
			case "DUMPCONFIG", "O":
				p.dumpConfig()
			case "PANIC":
				_ = make([]any, 0)[1]
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

func (p *PassPersist) dumpConfig() {
	b, err := json.MarshalIndent(map[string]any{
		"base-oid":     p.baseOID,
		"refresh-rate": p.refreshRate,
	}, "", "   ")
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(b))
}

func (p *PassPersist) overrideFromEnv() {
	if val, ok := os.LookupEnv("PASSPERSIST_BASE_OID"); ok {
		if o, err := NewOID(val); err == nil {
			slog.Info("overriding base OID from env", "was", p.baseOID.String(), "now", o.String())
			p.baseOID = o
		}
	}

	if val, ok := os.LookupEnv("PASSPERSIST_REFRESH_RATE"); ok {
		if r, err := time.ParseDuration(val); err == nil {
			slog.Info("overriding refresh rate from env", "was", p.refreshRate, "now", r)
			p.refreshRate = r
		}
	}
}

func (p *PassPersist) update(ctx context.Context, callback func(*PassPersist)) {

	for {
		select {
		case <-ctx.Done():
			return
		default:
			timer := time.NewTimer(p.refreshRate)

			callback(p)
			p.cache.Commit()

			<-timer.C
		}
	}
}

func (p *PassPersist) get(oid OID) *VarBind {
	slog.Debug("getting oid", "oid", oid.String())
	return p.cache.Get(oid)
}

func (p *PassPersist) getNext(oid OID) *VarBind {
	return p.cache.GetNext(oid)
}

func watchStdin(ctx context.Context, input chan<- string, done chan<- bool) {

	scanner := bufio.NewScanner(os.Stdin)

	defer func() {
		done <- true
	}()

	for scanner.Scan() {

		select {
		case <-ctx.Done():
			return
		default:
			line := scanner.Text()
			slog.Debug("got user input", "input", line)
			input <- line
		}
	}

	if err := scanner.Err(); err != nil {
		if err != io.EOF {
			slog.Error("scanner encountered an error", slog.Any("error", err.Error()))
			os.Exit(1)
		}
	}
}

func convertAndValidateOID(oid string, baseOID OID) (OID, error) {
	o, err := NewOID(oid)

	if err != nil {
		return OID{}, fmt.Errorf("failed to load oid: %s", oid)
	}

	if !o.Contains(baseOID) {
		return o, fmt.Errorf("oid '%s' does not contain base OID '%s'", o.String(), baseOID.String())
	}

	return o, nil
}
