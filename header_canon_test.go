package uhttp

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestCanon(t *testing.T) {
	type testcase struct {
		desc     string
		orig     string
		retain   []string
		expected string
	}

	cases := []testcase{
		{"should do nothing", "One: 1\r\nTwo: 2\r\n\r\n", []string{"One", "Two"}, "One: 1\r\nTwo: 2\r\n\r\n"},
		{"strip first", "One: 1\r\nTwo: 2\r\n\r\n", []string{"Two"}, "Two: 2\r\n\r\n"},
		{"strip last", "One: 1\r\nTwo: 2\r\n\r\n", []string{"One"}, "One: 1\r\n\r\n"},
		{"strip all", "One: 1\r\nTwo: 2\r\n\r\n", []string{}, "\r\n"},
		{"case change", "One: 1\r\nTwo: 2\r\n\r\n", []string{"oNE", "tWO"}, "oNE: 1\r\ntWO: 2\r\n\r\n"},
	}

	for _, c := range cases {
		var b bytes.Buffer

		fn := func(name string) string {
			for _, s := range c.retain {
				if strings.ToUpper(s) == strings.ToUpper(name) {
					return s
				}
			}
			return ""
		}
		s := newHeaderCanon(fn, &b)
		_, err := io.WriteString(s, c.orig)
		if err != nil {
			t.Errorf("%s: did not expect error, got %v", c.desc, err)
			continue
		}
		err = s.Close()
		if err != nil {
			t.Errorf("%s: did not expect error, got %v", c.desc, err)
			continue
		}
		actual := string(b.Bytes())
		if actual != c.expected {
			t.Errorf("%s: expected %q, got %q", c.desc, c.expected, actual)
		}
	}
}
