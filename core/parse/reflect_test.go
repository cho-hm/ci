package parse

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_filterComment(t *testing.T) {
	strs := []string{
		`KEY:"#comment`,
		`KEY:"#not-comment"`,
		`KEY:"#comment'`,
		`KEY:'#comment"`,
		`KEY:SOME' #VALUE`,
		`KEY:SOME"VAL#U"E`,
		`KEY:SOME VAL#U"E`,
		`KEY:SOME VALU"E`,
		`KEY:"SO'M#E V"AL'UE`,
	}

	expect := []string{
		`KEY:"`,
		`KEY:"#not-comment"`,
		`KEY:"`,
		`KEY:'`,
		`KEY:SOME'`,
		`KEY:SOME"VAL#U"E`,
		`KEY:SOME VAL`,
		`KEY:SOME VALU"E`,
		`KEY:"SO'M#E V"AL'UE`,
	}

	reader := strings.NewReader(strings.Join(strs, "\n"))

	scan := bufio.NewScanner(reader)
	scan.Split(bufio.ScanLines)
	ret := filterComment(scan)

	for i, r := range ret {
		if expect[i] != r {
			t.Errorf("case %d: got %q, want %q", i, r, expect[i])
		}
	}
}

func Test_filterComment_skipsBlankLines(t *testing.T) {
	input := "key1=val1\n\n  \n\nkey2=val2\n"
	scan := bufio.NewScanner(strings.NewReader(input))
	scan.Split(bufio.ScanLines)

	ret := filterComment(scan)

	for _, line := range ret {
		if strings.TrimSpace(line) == "" {
			t.Error("blank line should have been filtered out")
		}
	}
	if len(ret) != 2 {
		t.Errorf("expected 2 lines after filtering blanks, got %d", len(ret))
	}
}

func Test_filterComment_handlesCRLF(t *testing.T) {
	input := "key1=val1\r\nkey2=val2\r\n"
	scan := bufio.NewScanner(strings.NewReader(input))
	scan.Split(bufio.ScanLines)

	ret := filterComment(scan)

	for _, line := range ret {
		if strings.ContainsRune(line, '\r') {
			t.Errorf("line should not contain \\r: %q", line)
		}
	}
}

func Test_stripQuotes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"hello"`, `hello`},
		{`'hello'`, `hello`},
		{`"hello'`, `"hello'`},
		{`hello`, `hello`},
		{`""`, ``},
		{`"`, `"`},
	}

	for _, tt := range tests {
		got := stripQuotes(tt.input)
		if got != tt.want {
			t.Errorf("stripQuotes(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func Test_resolveValueByTag(t *testing.T) {
	content := "  key.one = \"value1\" \nkey.two='value2'\n\n# comment line\nkey.three = has=equals\nno-equals-line\n  \r\n"
	dir := t.TempDir()
	propFile := filepath.Join(dir, "test.properties")
	if err := os.WriteFile(propFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	type target struct {
		One   string `pkey:"key.one"`
		Two   string `pkey:"key.two"`
		Three string `pkey:"key.three"`
	}
	var obj target
	if err := resolveValueByTag(propFile, &obj); err != nil {
		t.Fatal(err)
	}

	if obj.One != "value1" {
		t.Errorf("One = %q, want %q", obj.One, "value1")
	}
	if obj.Two != "value2" {
		t.Errorf("Two = %q, want %q", obj.Two, "value2")
	}
	if obj.Three != "has=equals" {
		t.Errorf("Three = %q, want %q", obj.Three, "has=equals")
	}
}
