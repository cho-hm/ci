package parse

import (
	"bufio"
	"os"
	"reflect"
	"strings"
)

func resolveValueByTag(pathStr string, obj any) error {
	file, err := os.Open(pathStr)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var actual []string = filterComment(scanner)
	var maps = make(map[string]string)
	for _, s := range actual {
		split := strings.SplitN(s, "=", 2)
		if len(split) < 2 {
			continue
		}
		k := strings.TrimSpace(split[0])
		v := strings.TrimSpace(split[1])
		v = stripQuotes(v)
		maps[k] = v
	}

	setByReflect(maps, obj)

	return nil
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func setByReflect(maps map[string]string, obj any) {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		panic("not ptr")
	}

	v = v.Elem()

	if v.Kind() != reflect.Struct {
		panic("not struct")
	}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		ft := t.Field(i)
		fv := v.Field(i)

		if !ft.IsExported() || !fv.CanSet() {
			continue
		}

		tagValue := ft.Tag.Get("pkey")
		if tagValue == "" {
			continue
		}
		v, ok := maps[tagValue]
		if !ok {
			continue
		}
		if fv.Kind() == reflect.String {
			fv.SetString(v)
		}
	}
}

func filterComment(scanner *bufio.Scanner) []string {

	var ret []string
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, "#") {
			dq := false
			sq := false

			st := make([]int, 0, len(line))
			for i, c := range line {
				if c == '"' {
					st = resolve('"', &dq, line, st, i)
				} else if c == '\'' {
					st = resolve('\'', &sq, line, st, i)
				} else if c == '#' {
					st = append(st, i)
				}
			}

			for _, pos := range st {
				if line[pos] == '#' {
					line = strings.TrimRight(line[:pos], " \t")
					break
				}
			}
		}
		ret = append(ret, line)
	}
	return ret
}

func resolve(quote rune, qFlag *bool, line string, st []int, i int) []int {
	if *qFlag {
		removeTo(quote, line, &st)
		*qFlag = false
	} else {
		*qFlag = true
		st = append(st, i)
	}
	return st
}

func removeTo(token rune, orin string, st *[]int) {
	for i := len(*st) - 1; i >= 0; i-- {
		current := rune(orin[(*st)[i]])
		if current == token {
			*st = (*st)[:i]
			return
		}
	}
}
