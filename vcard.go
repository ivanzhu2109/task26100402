// Package vcard 提供 vCard 3.0 (UTF-8 文本) 联系人的离线读写能力。
package vcard

import "strings"

// Param 表示一个属性参数，保留多个值与重复出现的参数。
type Param struct {
	Name   string
	Values []string
}

// Property 表示一条 vCard 属性行（已展开折行）。
type Property struct {
	Group  string  // 组名前缀，无则为空
	Name   string  // 属性名，统一大写存储
	Params []Param // 参数，保留出现顺序与重复项
	Value  string  // 冒号之后的原始值（未反转义）
}

// Card 表示一张名片，保留属性顺序与同名属性的全部出现项。
type Card struct {
	Props []Property
}

// Get 返回首个匹配属性名（大小写不敏感）的属性，不存在时返回 nil。
func (c *Card) Get(name string) *Property {
	for i := range c.Props {
		if strings.EqualFold(c.Props[i].Name, name) {
			return &c.Props[i]
		}
	}
	return nil
}

// All 返回所有匹配属性名（大小写不敏感）的属性。
func (c *Card) All(name string) []*Property {
	var out []*Property
	for i := range c.Props {
		if strings.EqualFold(c.Props[i].Name, name) {
			out = append(out, &c.Props[i])
		}
	}
	return out
}

// Parse 解析 vCard 3.0 文本（接受 CRLF 或 LF，允许末尾无换行）。
// 只有独立逻辑行 BEGIN:VCARD / END:VCARD（大小写不敏感）是名片边界。
func Parse(data []byte) ([]*Card, error) {
	var cards []*Card
	var cur *Card
	for _, line := range unfoldLines(data) {
		if strings.EqualFold(line, "BEGIN:VCARD") {
			cur = &Card{}
			continue
		}
		if strings.EqualFold(line, "END:VCARD") {
			if cur != nil {
				cards = append(cards, cur)
				cur = nil
			}
			continue
		}
		if line == "" {
			continue
		}
		if cur == nil {
			continue // 名片边界之外的内容忽略
		}
		p, err := parseLine(line)
		if err != nil {
			return nil, err
		}
		cur.Props = append(cur.Props, p)
	}
	return cards, nil
}

// Marshal 将名片序列化为 vCard 3.0 文本，统一 CRLF，
// 每条物理行不超过 75 字节（不计行末 CRLF）。
func Marshal(cards []*Card) []byte {
	var out []byte
	for _, c := range cards {
		out = append(out, foldLine("BEGIN:VCARD")...)
		for _, p := range c.Props {
			out = append(out, foldLine(writeProperty(p))...)
		}
		out = append(out, foldLine("END:VCARD")...)
	}
	return out
}
