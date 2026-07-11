package validate

import "testing"

func TestValidatePositiveID(t *testing.T) {
	cases := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{"positivo", 1, false},
		{"grande", 1 << 40, false},
		{"cero", 0, true},
		{"negativo", -5, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidatePositiveID("field", c.id)
			if c.wantErr && err == nil {
				t.Fatalf("id=%d: esperaba error", c.id)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("id=%d: error inesperado: %v", c.id, err)
			}
		})
	}
}
