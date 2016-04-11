package input

import "testing"

func Test_Init(t *testing.T) {

	var p Input

	mismatched := func(rtrnd string, intnd string, intndA string) {

		if intndA != "" {

			t.Fatalf("Parser format %v does not match the intended format %v.", rtrnd, intnd)

		} else {

			t.Fatalf("Parser format %v does not match the indended format %v, which is equal to %v.", rtrnd, intndA, intnd)

		}

	}

	for i, f := range FORMATS_BY_NAME {

		p = NewParser(f)

		if p.Format != FORMATS_BY_STANDARD[i] {

			mismatched(p.Format, f, FORMATS_BY_STANDARD[i])

		}

	}

	for _, f := range FORMATS_BY_STANDARD {

		p = NewParser(f)

		if p.Format != f {

			mismatched(p.Format, f, "")

		}

	}

	p = NewParser("unknown-format")

	if p.Format != "ecma404" {

		mismatched(p.Format, "ecma404", "")

	}

}
