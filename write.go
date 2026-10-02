package vcard

import (
	"strings"
	"unicode/utf8"
)

// maxLineBytes 是每条物理行的最大字节数，不计行末 CRLF，
// 续行前缀空格计入限额。
const maxLineBytes = 75

// foldLine 将一条逻辑行按 75 字节限制折行，统一 CRLF，
// 续行前缀为一个空格，不得从 UTF-8 字符内部折行。
func foldLine(line string) []byte {
	var out []byte
	limit := maxLineBytes // 首行限额
	for len(line) > 0 {
		n := limit
		if len(line) < n {
			n = len(line)
		}
		// 不得切断多字节 UTF-8 字符
		for n > 0 && !utf8.ValidString(line[:n]) {
			n--
		}
		if n == 0 {
			n = 1 // 防御：至少前进一个字节
		}
		out = append(out, line[:n]...)
		out = append(out, '\r', '\n')
		line = line[n:]
		if len(line) > 0 {
			out = append(out, ' ')
			limit = maxLineBytes - 1 // 续行前缀空格计入限额
		}
	}
	return out
}

// writeProperty 将属性渲染为逻辑行文本（不含折行）。
func writeProperty(p Property) string {
	var b strings.Builder
	if p.Group != "" {
		b.WriteString(p.Group)
		b.WriteByte('.')
	}
	b.WriteString(p.Name)
	for _, param := range p.Params {
		b.WriteByte(';')
		b.WriteString(param.Name)
		b.WriteByte('=')
		for i, v := range param.Values {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(quoteParamValue(v))
		}
	}
	b.WriteByte(':')
	b.WriteString(p.Value)
	return b.String()
}

// quoteParamValue 在参数值含有分隔符或引号时加双引号。
func quoteParamValue(v string) string {
	if strings.ContainsAny(v, ";:,\"") {
		return `"` + v + `"`
	}
	return v
}
