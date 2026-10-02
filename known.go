package vcard

import "strings"

// 已知字段的结构化访问。解析时只有未被转义的分隔符划分结构，
// 写回时重新转义，保证结构与值一致。

// Text 返回反转义后的文本值（FN、TEL、EMAIL、NOTE 等）。
func (p *Property) Text() string { return unescapeText(p.Value) }

// SetText 设置文本值并重新转义。
func (p *Property) SetText(s string) { p.Value = escapeText(s) }

// Components 按未转义的分号切分并逐项反转义（N、ORG、ADR）。
// 空分量保留。
func (p *Property) Components() []string {
	parts := splitEscaped(p.Value, ';')
	for i := range parts {
		parts[i] = unescapeText(parts[i])
	}
	return parts
}

// ComponentsPadded 同 Components，不足 width 的尾部补空分量。
func (p *Property) ComponentsPadded(width int) []string {
	parts := p.Components()
	for len(parts) < width {
		parts = append(parts, "")
	}
	return parts
}

// SetComponents 设置分量并重新转义。
func (p *Property) SetComponents(parts []string) {
	esc := make([]string, len(parts))
	for i, s := range parts {
		esc[i] = escapeText(s)
	}
	p.Value = strings.Join(esc, ";")
}

// Items 按未转义的逗号切分并逐项反转义（CATEGORIES）。
// 空列表项保留。
func (p *Property) Items() []string {
	parts := splitEscaped(p.Value, ',')
	for i := range parts {
		parts[i] = unescapeText(parts[i])
	}
	return parts
}

// SetItems 设置列表项并重新转义。
func (p *Property) SetItems(items []string) {
	esc := make([]string, len(items))
	for i, s := range items {
		esc[i] = escapeText(s)
	}
	p.Value = strings.Join(esc, ",")
}

// Structured 返回分量化的多值结构（N 5 个分量、ADR 7 个分量），
// 每个分量是逗号分隔的多值列表，不足的尾部补空。
func (p *Property) Structured(width int) [][]string {
	out := make([][]string, 0, width)
	for _, comp := range splitEscaped(p.Value, ';') {
		var vals []string
		for _, v := range splitEscaped(comp, ',') {
			vals = append(vals, unescapeText(v))
		}
		if len(vals) == 0 {
			vals = []string{""}
		}
		out = append(out, vals)
	}
	for len(out) < width {
		out = append(out, []string{""})
	}
	return out
}

// SetStructured 设置分量化的多值结构并重新转义。
func (p *Property) SetStructured(comps [][]string) {
	parts := make([]string, len(comps))
	for i, vals := range comps {
		esc := make([]string, len(vals))
		for j, v := range vals {
			esc[j] = escapeText(v)
		}
		parts[i] = strings.Join(esc, ",")
	}
	p.Value = strings.Join(parts, ";")
}

// N 分量数与 ADR 分量数。
const (
	NComponents   = 5
	ADRComponents = 7
)
