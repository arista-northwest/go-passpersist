package passpersist

// taken from: https://github.com/k-sone/snmpgo/blob/master/variables.go

import (
	"bytes"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type InvalidOIDErr struct {
	Value   string
	Message string
}

func (e *InvalidOIDErr) Error() string {
	return fmt.Sprintf("Invalid OID '%s': %s", e.Value, e.Message)
}

type OID struct {
	Value asn1.ObjectIdentifier
}

func (o *OID) EnvDecode(value string) error {
	*o = MustNewOID(value)
	return nil
}

func (v OID) String() string {
	return v.Value.String()
}

func (v OID) Type() string {
	return "OID"
}

func (v OID) Marshal() ([]byte, error) {
	return asn1.Marshal(v.Value)
}

func (v OID) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

// func (v *OID) Unmarshal(b []byte) (rest []byte, err error) {
// 	var i asn1.ObjectIdentifier
// 	rest, err = ber.Unmarshal(b, &i)
// 	if err == nil {
// 		v.Value = i
// 	}
// 	return
// }

// Returns true if this OID contains the specified OID
func (v OID) Contains(o OID) bool {
	if len(v.Value) < len(o.Value) {
		return false
	}
	for i := 0; i < len(o.Value); i++ {
		if v.Value[i] != o.Value[i] {
			return false
		}
	}
	return true
}

// Returns 0 this OID is equal to the specified OID,
// -1 this OID is lexicographically less than the specified OID,
// 1 this OID is lexicographically greater than the specified OID
func (v OID) Compare(o OID) int {
	vl := len(v.Value)
	ol := len(o.Value)

	for i := 0; i < vl; i++ {
		if ol <= i || v.Value[i] > o.Value[i] {
			return 1
		} else if v.Value[i] < o.Value[i] {
			return -1
		}
	}

	if ol == vl {
		return 0
	}

	return -1
}

// Returns true if this OID is same the specified OID
func (v OID) Equal(o OID) bool {
	return v.Value.Equal(o.Value)
}

func (v OID) StartsWith(o OID) bool {
	if len(v.Value) >= len(o.Value) {
		return v.Value[:len(o.Value)].Equal(o.Value)
	}
	return false
}

// Returns OID with additional sub-ids
func (v OID) Append(subs []int) (OID, error) {
	buf := bytes.NewBufferString(v.String())
	for _, i := range subs {
		buf.WriteString(".")
		buf.WriteString(strconv.Itoa(i))
	}
	return NewOID(buf.String())
}

func (v OID) MustAppend(subs []int) OID {
	o, err := v.Append(subs)
	if err != nil {
		slog.Error("failed to append subs", slog.Any("error", err))
		os.Exit(1)
	}
	return o
}

func NewOID(s string) (OID, error) {
	subids := strings.Split(s, ".")

	// if there is a leadong dot, the first element will be empty
	if subids[0] == "" {
		subids = subids[1:]
	}

	// ISO/IEC 8825 Section 8.19.4
	if len(subids) < 2 {
		return OID{}, &InvalidOIDErr{s, "The first and second sub-identifier is required"}
	}

	// RFC2578 Section 3.5
	if len(subids) > 128 {
		return OID{}, &InvalidOIDErr{s, "The sub-identifiers in an OID is up to 128"}
	}

	oid := make(asn1.ObjectIdentifier, len(subids))
	for i, v := range subids {
		val, err := strconv.Atoi(v)
		if err != nil || val < 0 || int64(val) > math.MaxUint32 {
			return OID{}, &InvalidOIDErr{s, fmt.Sprintf("The sub-identifiers is range %d..%d", 0, int64(math.MaxUint32))}
		}
		oid[i] = val
	}

	if len(oid) > 0 && oid[0] > 2 {
		return OID{}, &InvalidOIDErr{s, "The first sub-identifier is range 0..2"}
	}

	if oid[0] < 2 && oid[1] >= 40 {
		return OID{}, &InvalidOIDErr{s, "The second sub-identifier is range 0..39"}
	}

	return OID{oid}, nil
}

// MustNewOID is like NewOID but panics if argument cannot be parsed
func MustNewOID(s string) OID {
	oid, err := NewOID(s)
	if err != nil {
		slog.Error("failed to create new OID", slog.Any("error", err))
		os.Exit(1)
	}

	return oid
}

type OIDs []OID

// Sort a OID list
func (o OIDs) Sort() OIDs {
	c := make(OIDs, len(o))
	copy(c, o)
	sort.Sort(sortableOIDs{c})
	return c
}

func (o OIDs) uniq(comp func(a, b OID) bool) OIDs {
	var before OID
	c := make(OIDs, 0, len(o))
	for _, oid := range o {
		if !comp(before, oid) {
			before = oid
			c = append(c, oid)
		}
	}
	return c
}

// Filter out adjacent OID list
func (o OIDs) Uniq() OIDs {
	return o.uniq(func(a, b OID) bool {
		return b.Equal(a)
	})
}

// Filter out adjacent OID list with the same prefix
func (o OIDs) UniqBase() OIDs {
	return o.uniq(func(a, b OID) bool {
		return b.Contains(a)
	})
}

type sortableOIDs struct {
	OIDs
}

func (o sortableOIDs) Len() int {
	return len(o.OIDs)
}

func (o sortableOIDs) Swap(i, j int) {
	o.OIDs[i], o.OIDs[j] = o.OIDs[j], o.OIDs[i]
}

func (o sortableOIDs) Less(i, j int) bool {
	return o.OIDs[i].Compare(o.OIDs[j]) < 1
}

func NewOIDs(s []string) (oids OIDs, err error) {
	for _, l := range s {
		o, e := NewOID(l)
		if e != nil {
			return nil, e
		}
		oids = append(oids, o)
	}
	return
}
