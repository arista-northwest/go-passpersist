package passpersist

import "testing"

func TestCacheSet(t *testing.T) {
	c := NewCache()
	varBinds := []*VarBind{
		{
			OID:       MustNewOID("1.3.6.1.4.1.30065.4.224.255.0"),
			ValueType: "STRING",
			Value:     typedValue{Value: &StringVal{Value: "TEST"}},
		},
	}

	for _, vb := range varBinds {
		c.Set(vb)

	}

	c.Commit()

	for _, vb := range varBinds {
		if !c.Get(vb.OID).OID.Equal(vb.OID) {
			t.Errorf("var binds do not match")
		}
	}

	c.Dump()
}
