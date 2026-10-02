package vcard

import "strings"

// unescapeText 解码 vCard 文本值中的反斜杠转义：
// \\ 表示反斜杠，\, 表示逗号，\; 表示分号，\n 与 \N 表示换行。
// 其他反斜杠序列按原样保留。
func unescapeText(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		switch s[i+1] {
		case '\\':
			b.WriteByte('\\')
		case ',':
			b.WriteByte(',')
		case ';':
			b.WriteByte(';')
		case 'n', 'N':
			b.WriteByte('\n')
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i+1])
		}
		i++
	}
	return b.String()
}

// escapeText 对 vCard 文本值进行反斜杠转义：
// 反斜杠、逗号、分号转义，换行写作 \n。
func escapeText(s string) string {
	if !strings.ContainsAny(s, "\\,;\n\r") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			b.WriteString(`\\`)
		case ',':
			b.WriteString(`\,`)
		case ';':
			b.WriteString(`\;`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			// 丢弃回车，避免与 CRLF 行尾混淆
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
