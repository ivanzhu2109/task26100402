package vcard

import "strings"

// Name 表示 N 属性的 5 个分量，每个分量可含多个值。
type Name struct {
	Family, Given, Additional, Prefix, Suffix []string
}

// Address 表示 ADR 属性的 7 个分量，每个分量可含多个值。
type Address struct {
	POBox, Extended, Street, Locality, Region, PostalCode, Country []string
}

// splitUnescaped 按未转义的分隔符切分并逐项解码，保留空项。
func splitUnescaped(s string, sep byte) []string {
	raw := splitRaw(s, sep)
	for i := range raw {
		raw[i] = unescapeText(raw[i])
	}
	return raw
}

// splitRaw 按未转义的分隔符切分但不解码，保留各项的转义符。
func splitRaw(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			i++ // 跳过被转义的下一个字节
			continue
		}
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

// joinEscaped 逐项转义后用分隔符连接。
func joinEscaped(items []string, sep byte) string {
	esc := make([]string, len(items))
	for i, v := range items {
		esc[i] = escapeText(v)
	}
	return strings.Join(esc, string(sep))
}

// parseStructured 把结构值切为固定数量的分量，不足补空，超出并入末尾；
// 每个分量再按逗号切为多值。
func parseStructured(value string, count int) [][]string {
	comps := splitRaw(value, ';')
	out := make([][]string, count)
	for i := range out {
		if i < len(comps) {
			out[i] = splitUnescaped(comps[i], ',')
		} else {
			out[i] = []string{""}
		}
	}
	return out
}

// buildStructured 把固定数量的分量编码回结构值。
func buildStructured(comps [][]string) string {
	parts := make([]string, len(comps))
	for i, c := range comps {
		parts[i] = joinEscaped(c, ',')
	}
	return strings.Join(parts, ";")
}

// find 返回首个匹配属性名的下标（大小写不敏感），不存在返回 -1。
func (c *Contact) find(name string) int {
	for i := range c.Props {
		if strings.EqualFold(c.Props[i].Name, name) {
			return i
		}
	}
	return -1
}

// Get 返回首个匹配属性名的属性（大小写不敏感），不存在时返回 nil。
func (c *Contact) Get(name string) *Property {
	if i := c.find(name); i >= 0 {
		return &c.Props[i]
	}
	return nil
}

// GetAll 返回全部匹配属性名的属性，保持原有顺序。
func (c *Contact) GetAll(name string) []Property {
	var out []Property
	for _, p := range c.Props {
		if strings.EqualFold(p.Name, name) {
			out = append(out, p)
		}
	}
	return out
}

// setValue 设置首个匹配属性的值；不存在则追加新属性。
func (c *Contact) setValue(name, value string) {
	if i := c.find(name); i >= 0 {
		c.Props[i].Value = value
		return
	}
	c.Props = append(c.Props, Property{Name: strings.ToUpper(name), Value: value})
}

// FN 返回 FN 的解码值。
func (c *Contact) FN() string {
	if p := c.Get("FN"); p != nil {
		return unescapeText(p.Value)
	}
	return ""
}

// SetFN 设置 FN 的值；不存在则追加。
func (c *Contact) SetFN(v string) { c.setValue("FN", escapeText(v)) }

// Note 返回 NOTE 的解码值。
func (c *Contact) Note() string {
	if p := c.Get("NOTE"); p != nil {
		return unescapeText(p.Value)
	}
	return ""
}

// SetNote 设置 NOTE 的值；不存在则追加。
func (c *Contact) SetNote(v string) { c.setValue("NOTE", escapeText(v)) }

// Name 返回 N 的结构化值。
func (c *Contact) Name() Name {
	var n Name
	p := c.Get("N")
	if p == nil {
		return n
	}
	comps := parseStructured(p.Value, 5)
	n.Family, n.Given, n.Additional, n.Prefix, n.Suffix = comps[0], comps[1], comps[2], comps[3], comps[4]
	return n
}

// SetName 设置 N 的结构化值。
func (c *Contact) SetName(n Name) {
	c.setValue("N", buildStructured([][]string{n.Family, n.Given, n.Additional, n.Prefix, n.Suffix}))
}

// Org 返回 ORG 的分量列表。
func (c *Contact) Org() []string {
	if p := c.Get("ORG"); p != nil {
		return splitUnescaped(p.Value, ';')
	}
	return nil
}

// SetOrg 设置 ORG 的分量列表。
func (c *Contact) SetOrg(parts []string) { c.setValue("ORG", joinEscaped(parts, ';')) }

// Categories 返回 CATEGORIES 的列表值。
func (c *Contact) Categories() []string {
	if p := c.Get("CATEGORIES"); p != nil {
		return splitUnescaped(p.Value, ',')
	}
	return nil
}

// SetCategories 设置 CATEGORIES 的列表值。
func (c *Contact) SetCategories(items []string) { c.setValue("CATEGORIES", joinEscaped(items, ',')) }

// Telephones 返回全部 TEL 属性。
func (c *Contact) Telephones() []Property { return c.GetAll("TEL") }

// Emails 返回全部 EMAIL 属性。
func (c *Contact) Emails() []Property { return c.GetAll("EMAIL") }

// Addresses 返回全部 ADR 的结构化值。
func (c *Contact) Addresses() []Address {
	props := c.GetAll("ADR")
	out := make([]Address, len(props))
	for i, p := range props {
		out[i] = addressFromComponents(parseStructured(p.Value, 7))
	}
	return out
}

// SetAddress 设置第 i 个 ADR 的结构化值，下标越界返回 false。
func (c *Contact) SetAddress(i int, a Address) bool {
	idx := -1
	for j := range c.Props {
		if !strings.EqualFold(c.Props[j].Name, "ADR") {
			continue
		}
		idx++
		if idx == i {
			c.Props[j].Value = buildStructured(addressComponents(a))
			return true
		}
	}
	return false
}

func addressFromComponents(comps [][]string) Address {
	return Address{
		POBox: comps[0], Extended: comps[1], Street: comps[2],
		Locality: comps[3], Region: comps[4], PostalCode: comps[5], Country: comps[6],
	}
}

func addressComponents(a Address) [][]string {
	return [][]string{a.POBox, a.Extended, a.Street, a.Locality, a.Region, a.PostalCode, a.Country}
}
