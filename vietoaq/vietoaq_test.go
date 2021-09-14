package vietoaq

import (
	"testing"
)

var (
	examples = map[string]string{
		``: ``,
		`bbb`: `bbb`,
		`jảq hủı óq`: `jam huin xob`,
		`ýhō`: `xyphor`,
		`gı'aq`: `gixaq`,
		`gï'aq`: `gixxaq`,
		`jảq'a`: `jamxa`,
		`gï aq`: `gix xaq`,
		`aq'aq aq`: `xaqxaq xaq`,
	}
)

func TestVietoaq(t *testing.T) {
	for regular, vietoaq := range examples {
		vietoaq_ := To(regular)
		if vietoaq_ != vietoaq {
			t.Errorf("  to: %s -> %s != %s", regular, vietoaq_, vietoaq)
		}
		regular_ := From(vietoaq)
		if regular_ != regular {
			t.Errorf("from: %s -> %s != %s", vietoaq, regular_, regular)
		}
	}
}

