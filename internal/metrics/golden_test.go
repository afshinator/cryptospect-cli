package metrics

import (
	"bytes"
	"testing"
)

func TestNormaliseJSON_KeyOrdering(t *testing.T) {
	a := `{"z":1,"a":2}`
	b := `{"a":2,"z":1}`

	na, err := NormaliseJSON([]byte(a))
	if err != nil {
		t.Fatalf("normalise a: %v", err)
	}
	nb, err := NormaliseJSON([]byte(b))
	if err != nil {
		t.Fatalf("normalise b: %v", err)
	}
	if !bytes.Equal(na, nb) {
		t.Errorf("normalised forms differ:\n%s\n%s", na, nb)
	}
}

func TestNormaliseJSON_InvalidInput(t *testing.T) {
	_, err := NormaliseJSON([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
