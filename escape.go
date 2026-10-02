package vcard

import "strings"

// unescapeText 解码已知文本值中的反斜杠转义：
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

// escapeText 对已知文本值重新转义，保证往返一致。
func escapeText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
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
			// 单独的 CR 不是合法值内容，规整为换行
			b.WriteString(`\n`)
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// splitEscaped 按未被转义的分隔符切分，保留空项与空列表项。
func splitEscaped(s string, sep byte) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++ // 跳过转义序列，其中的分隔符不生效
			continue
		}
		if s[i] == sep {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}
