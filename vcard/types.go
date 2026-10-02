// Package vcard 提供 vCard 3.0（UTF-8 文本）联系人的读写能力。
package vcard

// Param 表示一个属性参数，保留多个值与重复出现的参数。
type Param struct {
	Name   string
	Values []string
}

// Property 表示一行 vCard 属性，保留组名前缀、参数与原始值。
type Property struct {
	Group  string
	Name   string
	Params []Param
	Value  string // 展开折行后的原始值（保留转义符）
}

// Contact 表示一张名片，属性按原顺序保留。
type Contact struct {
	Props []Property
}
