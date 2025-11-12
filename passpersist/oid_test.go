package passpersist

import (
	"fmt"
	"testing"
)

func TestNewOIDEmpty(t *testing.T) {
	o := MustNewOID("1.3")

	_, err := o.Append([]int{1, 2, 3, 4})
	if err != nil {
		t.Error(err)
	}

}

func TestOIDAppend(t *testing.T) {
	o := MustNewOID("1.2")

	o, _ = o.Append([]int{3, 4})
	o, _ = o.Append([]int{5, 6})
	o, _ = o.Append([]int{7, 8})

	fmt.Println(o.String())
}

func TestInsertSorted(t *testing.T) {
	oids := OIDs{
		MustNewOID("1.2.3.4.7"),
		MustNewOID("1.2.3.4.1"),
		MustNewOID("1.2.3.4.10"),
		MustNewOID("1.2.3.4.5"),
		MustNewOID("1.2.3.4.3"),
	}

	oids = append(oids, MustNewOID("1.2.3.4.6"))

	fmt.Println(oids.Sort())
	//oids := make(OIDs, 0, len(idx))
}

func TestOIDContains(t *testing.T) {
	base := MustNewOID("1.3.6.1.4.1.8072")
	oid := MustNewOID("1.3.6.1.4.1.8072.1")

	if !oid.Contains(base) {
		t.Errorf("expected oid '%s' to contain base '%s'", oid, base)
	}
}

func TestNewOIDs(t *testing.T) {
	oids, err := NewOIDs([]string{
		"1.2.3.4.5",
		"5.4.3.2.1",
	})

	if err != nil {
		t.Error(err)
	}

	if len(oids) != 2 {
		t.Errorf("expected 2 oids, got %d", len(oids))
	}
}
