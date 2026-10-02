package vcard

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Parse 从 io.Reader 读取并解析 vCard 3.0 文本，返回全部名片。
func Parse(r io.Reader) ([]*Contact, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parseBytes(data)
}

// ParseFile 从本地文件读取并解析 vCard 3.0 文本。
func ParseFile(path string) ([]*Contact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseBytes(data)
}

// parseBytes 在字节层面展开折行（允许折行落在 UTF-8 字符内部），
// 再按逻辑行解析属性。
func parseBytes(data []byte) ([]*Contact, error) {
	lines := unfold(data)
	var contacts []*Contact
	var cur *Contact
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		upper := strings.ToUpper(string(line))
		if upper == "BEGIN:VCARD" {
			if cur != nil {
				return nil, fmt.Errorf("第 %d 行：VCARD 未结束又出现 BEGIN", i+1)
			}
			cur = &Contact{}
			continue
		}
		if upper == "END:VCARD" {
			if cur == nil {
				return nil, fmt.Errorf("第 %d 行：END:VCARD 没有对应的 BEGIN", i+1)
			}
			contacts = append(contacts, cur)
			cur = nil
			continue
		}
		if cur == nil {
			return nil, fmt.Errorf("第 %d 行：属性出现在 VCARD 边界之外", i+1)
		}
		prop, err := parseProperty(string(line))
		if err != nil {
			return nil, fmt.Errorf("第 %d 行：%w", i+1, err)
		}
		cur.Props = append(cur.Props, prop)
	}
	if cur != nil {
		return nil, fmt.Errorf("文件结束：VCARD 缺少 END:VCARD")
	}
	return contacts, nil
}

// unfold 把物理行序列合并为逻辑行：换行后紧跟空格或制表符表示续行，
// 仅移除换行及紧随其后的一个空白字符。接受 CRLF 或 LF，
// 允许最后一行没有换行符。
func unfold(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	flush := func(end int) {
		line := data[start:end]
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && len(lines) > 0 {
			lines[len(lines)-1] = append(lines[len(lines)-1], line[1:]...)
		} else {
			lines = append(lines, line)
		}
	}
	for i := 0; i < len(data); i++ {
		if data[i] != '\n' {
			continue
		}
		end := i
		if end > start && data[end-1] == '\r' {
			end--
		}
		flush(end)
		start = i + 1
	}
	if start < len(data) {
		flush(len(data))
	}
	return lines
}

// parseProperty 解析一条逻辑行：可选组名前缀、属性名、参数与值。
// 值从参数引号之外的第一个冒号开始；引号内的分号、冒号、逗号均为内容。
func parseProperty(line string) (Property, error) {
	colon := indexOutsideQuotes(line, ':')
	if colon < 0 {
		return Property{}, fmt.Errorf("属性行缺少冒号：%q", line)
	}
	head, value := line[:colon], line[colon+1:]

	// 按引号外的分号切分头部：第一段是 [组.]名，其余是参数。
	segs := splitOutsideQuotes(head, ';')
	namePart := segs[0]
	prop := Property{Value: value}
	if dot := strings.IndexByte(namePart, '.'); dot >= 0 {
		prop.Group = namePart[:dot]
		prop.Name = namePart[dot+1:]
	} else {
		prop.Name = namePart
	}
	if prop.Name == "" {
		return Property{}, fmt.Errorf("属性名为空：%q", line)
	}
	for _, seg := range segs[1:] {
		eq := indexOutsideQuotes(seg, '=')
		if eq < 0 {
			// 无等号的参数（如 vCard 2.1 风格裸值），按单值参数保留。
			prop.Params = append(prop.Params, Param{Name: "", Values: []string{unquote(seg)}})
			continue
		}
		name := seg[:eq]
		rawVals := splitOutsideQuotes(seg[eq+1:], ',')
		vals := make([]string, len(rawVals))
		for i, v := range rawVals {
			vals[i] = unquote(v)
		}
		prop.Params = append(prop.Params, Param{Name: name, Values: vals})
	}
	return prop, nil
}

// indexOutsideQuotes 返回引号之外第一个目标字节的位置，不存在返回 -1。
func indexOutsideQuotes(s string, target byte) int {
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuote = !inQuote
		default:
			if !inQuote && s[i] == target {
				return i
			}
		}
	}
	return -1
}

// splitOutsideQuotes 按引号之外的分隔符切分，保留空段。
func splitOutsideQuotes(s string, sep byte) []string {
	var parts []string
	inQuote := false
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuote = !inQuote
		default:
			if !inQuote && s[i] == sep {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

// unquote 去掉成对的首尾双引号。
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
