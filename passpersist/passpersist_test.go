
package passpersist

import "testing"

func TestConvertAndValidateOID(t *testing.T) {
  _, err := convertAndValidateOID("1.3.6.1.4.1.8072.1", MustNewOID("1.3.6.1.4.1.8072"))
  if err != nil {
    t.Errorf("failed to parse: %s", err)
  }
}
