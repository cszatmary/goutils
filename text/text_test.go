package text_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cszatmary/goutils/text"
)

var expandVariablesTests = []struct {
	name string
	in   string
	out  string
}{
	{"empty", "", ""},
	{"no vars", "nothing to expand", "nothing to expand"},
	{"just a var", "${HOME}", "/home/foo"},
	{"var in middle", "start ${HOME} end", "start /home/foo end"},
	{"multiple vars", "foo ${first} bar ${second} baz", "foo abc bar def baz"},
	{"$", "$", "$"},
	{"$}", "$}", "$}"},
	{"${", "${", "${"},    // invalid syntax, will ignore
	{"${}", "${}", "${}"}, // invalid syntax, will ignore
	{"contains not vars", "start $HOME ${first} $$", "start $HOME abc $$"},
	{"non-alphanum var", "path: ${@env:HOME}", "path: $HOME"},
	{"side by side", "${first}${second}", "abcdef"},
}

func testMapping(name string) string {
	if strings.HasPrefix(name, "@env:") {
		return "$" + strings.TrimPrefix(name, "@env:")
	}
	switch name {
	case "HOME":
		return "/home/foo"
	case "first":
		return "abc"
	case "second":
		return "def"
	}
	return "UNKNOWN_VAR"
}

func TestExpandVariables(t *testing.T) {
	for _, tt := range expandVariablesTests {
		t.Run(tt.name, func(t *testing.T) {
			got := text.ExpandVariables([]byte(tt.in), testMapping)
			if string(got) != string(tt.out) {
				t.Errorf("got %q, want %q", got, tt.out)
			}
		})
	}
}

func TestExpandVariablesString(t *testing.T) {
	for _, tt := range expandVariablesTests {
		t.Run(tt.name, func(t *testing.T) {
			got := text.ExpandVariablesString(tt.in, testMapping)
			if got != tt.out {
				t.Errorf("got %q, want %q", got, tt.out)
			}
		})
	}
}

func TestVariableMapper(t *testing.T) {
	vm := text.NewVariableMapper(map[string]string{
		"HOME": "/home/foo",
		"foo":  "bar",
	})
	in := "${HOME}; ${missing1}; ${foo}; ${missing2}; ${missing1}; ${nope}"
	wantText := "/home/foo; ; bar; ; ; "
	got := text.ExpandVariablesString(in, vm.Map)
	if got != wantText {
		t.Errorf("got text %q, want %q", got, wantText)
	}

	wantMissing := []string{"missing1", "missing2", "nope"}
	if !reflect.DeepEqual(vm.Missing(), wantMissing) {
		t.Errorf("got missing %+v, want %+v", vm.Missing(), wantMissing)
	}
}

func BenchmarkExpandVariables(b *testing.B) {
	b.Run("no-op", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			text.ExpandVariables([]byte("noop noop noop noop"), func(s string) string { return "" })
		}
	})
	b.Run("multiple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			text.ExpandVariables([]byte("${foo} ${foo} ${foo} ${foo}"), func(s string) string { return "bar" })
		}
	})
}

func BenchmarkExpandVariablesString(b *testing.B) {
	b.Run("no-op", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			text.ExpandVariablesString("noop noop noop noop", func(s string) string { return "" })
		}
	})
	b.Run("multiple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			text.ExpandVariablesString("${foo} ${foo} ${foo} ${foo}", func(s string) string { return "bar" })
		}
	})
}
