package passpersist

import "testing"

func TestConvertAndValidateOID(t *testing.T) {
	_, err := convertAndValidateOID("1.3.6.1.4.1.8072.1", MustNewOID("1.3.6.1.4.1.8072"))
	if err != nil {
		t.Errorf("failed to parse: %s", err)
	}
}

func TestWithBaseOID(t *testing.T) {
	base := "1.3.6.1.4.1.8072"
	pp := NewPassPersist(WithBaseOID(MustNewOID(base)))
	if pp.baseOID.String() != base {
		t.Errorf("expected base OID to be %s, got %s", base, pp.baseOID)
	}
}
