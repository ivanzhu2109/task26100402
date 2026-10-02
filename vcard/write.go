package vcard

import (
	"bufio"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// maxLineBytes 是每条物理行的最大字节数（不含行末 CRLF，含续行前缀空格）。
const maxLineBytes = 75

// Write 将名片列表以 CRLF 换行、75 字节折行写出。
func Write(w io.Writer, contacts []*Contact) error {
	bw := bufio.NewWriter(w)
	for _, c := range contacts {
		if err := writeFolded(bw, "BEGIN:VCARD"); err != nil {
			return err
		}
		for _, p := range c.Props {
			if err := writeFolded(bw, propertyLine(p)); err != nil {
				return err
			}
		}
		if err := writeFolded(bw, "END:VCARD"); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// WriteFile 将名片列表写入本地文件。
func WriteFile(path string, contacts []*Contact) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return Write(f, contacts)
}

// propertyLine 把属性序列化为一条逻辑行，保留组名前缀与参数。
func propertyLine(p Property) string {
	var b strings.Builder
	if p.Group != "" {
		b.WriteString(p.Group)
		b.WriteByte('.')
	}
	b.WriteString(p.Name)
	for _, param := range p.Params {
		b.WriteByte(';')
		if param.Name != "" {
			b.WriteString(param.Name)
			b.WriteByte('=')
		}
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

// quoteParamValue 在参数值包含分隔符时加双引号。
func quoteParamValue(v string) string {
	if strings.ContainsAny(v, ";:,") {
		return `"` + v + `"`
	}
	return v
}

// writeFolded 写出一个逻辑行，按 75 字节限制折行，
// 续行以单个空格开头，且不从 UTF-8 字符内部断开。
func writeFolded(w io.Writer, line string) error {
	for _, physical := range foldLine(line) {
		if _, err := io.WriteString(w, physical); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\r\n"); err != nil {
			return err
		}
	}
	return nil
}

// foldLine 把逻辑行切成若干物理行：首行至多 75 字节，
// 续行为一个空格加至多 74 字节，折行点落在 UTF-8 字符边界上。
func foldLine(line string) []string {
	if len(line) <= maxLineBytes {
		return []string{line}
	}
	var out []string
	rest := line
	limit := maxLineBytes
	for len(rest) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(rest[cut]) {
			cut--
		}
		if cut == 0 {
			cut = limit // 极端情况下强制切断，避免死循环
		}
		out = append(out, rest[:cut])
		rest = rest[cut:]
		limit = maxLineBytes - 1 // 续行前缀空格计入限额
	}
	out = append(out, rest)
	for i := 1; i < len(out); i++ {
		out[i] = " " + out[i]
	}
	return out
}
