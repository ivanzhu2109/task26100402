package vcard

import (
	"fmt"
	"strings"
)

// parseLine 将一条逻辑行解析为 Property。
// 属性值从参数引号之外的第一个冒号开始；参数中的分号、冒号、
// 逗号若位于双引号内，均为内容，不参与分隔。
func parseLine(line string) (Property, error) {
	var p Property

	// 找引号之外的第一个冒号，冒号之后为原始值
	colon := -1
	inQuote := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inQuote = !inQuote
		case ':':
			if !inQuote {
				colon = i
			}
		}
		if colon >= 0 {
			break
		}
	}
	if colon < 0 {
		return p, fmt.Errorf("vcard: 属性行缺少冒号: %q", line)
	}
	head := line[:colon]
	p.Value = line[colon+1:]

	// head 按引号之外的分号切分：第一段是 [group.]name，其余为参数
	segs := splitOutsideQuote(head, ';')
	if segs[0] == "" {
		return p, fmt.Errorf("vcard: 属性名为空: %q", line)
	}
	if dot := strings.IndexByte(segs[0], '.'); dot >= 0 {
		p.Group = segs[0][:dot]
		p.Name = strings.ToUpper(segs[0][dot+1:])
	} else {
		p.Name = strings.ToUpper(segs[0])
	}
	if p.Name == "" {
		return p, fmt.Errorf("vcard: 属性名为空: %q", line)
	}

	for _, seg := range segs[1:] {
		if seg == "" {
			continue
		}
		name, values := seg, ""
		if eq := indexOutsideQuote(seg, '='); eq >= 0 {
			name, values = seg[:eq], seg[eq+1:]
		}
		param := Param{Name: strings.ToUpper(name)}
		for _, v := range splitOutsideQuote(values, ',') {
			param.Values = append(param.Values, unquote(v))
		}
		if len(param.Values) == 0 {
			param.Values = []string{""}
		}
		p.Params = append(p.Params, param)
	}
	return p, nil
}

// splitOutsideQuote 按引号之外的分隔符切分，保留空段与引号本身。
func splitOutsideQuote(s string, sep byte) []string {
	var out []string
	inQuote := false
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuote = !inQuote
		case sep:
			if !inQuote {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}

// indexOutsideQuote 返回引号之外第一个目标字节的位置，不存在返回 -1。
func indexOutsideQuote(s string, b byte) int {
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuote = !inQuote
		case b:
			if !inQuote {
				return i
			}
		}
	}
	return -1
}

// unquote 去除成对的首尾双引号。
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
