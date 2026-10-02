package vcard

import (
	"bytes"
	"strings"
	"testing"
)

// physicalLines 按 CRLF 切分输出，去掉末尾空段。
func physicalLines(t *testing.T, data []byte) []string {
	t.Helper()
	s := string(data)
	if !strings.HasSuffix(s, "\r\n") {
		t.Fatalf("输出未以 CRLF 结尾")
	}
	s = strings.TrimSuffix(s, "\r\n")
	if strings.Contains(s, "\n") && strings.Contains(s, "\r\n") {
		// ok，下面逐行检查
	}
	return strings.Split(s, "\r\n")
}

func checkLineLimit(t *testing.T, data []byte) {
	t.Helper()
	for i, line := range physicalLines(t, data) {
		if strings.ContainsRune(line, '\r') || strings.ContainsRune(line, '\n') {
			t.Fatalf("第 %d 行含裸换行", i+1)
		}
		if len(line) > maxLineBytes {
			t.Fatalf("第 %d 行 %d 字节，超过 %d 字节限制：%q", i+1, len(line), maxLineBytes, line)
		}
	}
}

// TestStructuredAndEscapes 覆盖：结构分隔符与转义标点混合、带引号参数、
// 中文折行（折行落在 UTF-8 字符内部）、组名前缀、空值与重复参数。
func TestStructuredAndEscapes(t *testing.T) {
	// 构造一条折行落在中文 UTF-8 字符内部的 NOTE。
	// 前缀 74 字节 ASCII，第 75 字节起是中文，折在第 75 字节处必然切断汉字。
	noteValue := strings.Repeat("a", 74) + "中文折行测试备注内容"
	folded := noteValue[:75] + "\r\n " + noteValue[75:]

	input := "BEGIN:VCARD\r\n" +
		"VERSION:3.0\r\n" +
		`N:张;三\,丰;小\;明;;博士` + "\r\n" +
		"FN:张三丰（博士）\r\n" +
		`ORG:某公司;研发\;测试部` + "\r\n" +
		`TEL;TYPE="WORK,VOICE";TYPE=CELL;X-EXTRA=a,b:+86-138-0000-0000` + "\r\n" +
		"TEL;TYPE=HOME:+86-10-12345678\r\n" +
		"EMAIL;TYPE=INTERNET:zhang@example.com\r\n" +
		`ADR;TYPE=HOME:;;中山\,路1号;北京;;100000;中国` + "\r\n" +
		`CATEGORIES:朋友\,同事,VIP,` + "\r\n" +
		"NOTE:" + folded + "\r\n" +
		"ITEM1.X-ABLABEL;X-FOO=bar:自定义标签\r\n" +
		"X-EMPTY:\r\n" +
		"END:VCARD\r\n"

	contacts, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("期望 1 张名片，得到 %d", len(contacts))
	}
	c := contacts[0]

	// 独立写出的预期字段
	if got := c.FN(); got != "张三丰（博士）" {
		t.Errorf("FN = %q", got)
	}
	name := c.Name()
	if len(name.Family) != 1 || name.Family[0] != "张" {
		t.Errorf("N.Family = %v", name.Family)
	}
	if len(name.Given) != 1 || name.Given[0] != "三,丰" {
		t.Errorf("N.Given = %v", name.Given)
	}
	if len(name.Additional) != 1 || name.Additional[0] != "小;明" {
		t.Errorf("N.Additional = %v", name.Additional)
	}
	if len(name.Prefix) != 1 || name.Prefix[0] != "" {
		t.Errorf("N.Prefix = %v（空分量丢失）", name.Prefix)
	}
	if len(name.Suffix) != 1 || name.Suffix[0] != "博士" {
		t.Errorf("N.Suffix = %v", name.Suffix)
	}
	if org := c.Org(); len(org) != 2 || org[0] != "某公司" || org[1] != "研发;测试部" {
		t.Errorf("ORG = %v", org)
	}
	tels := c.Telephones()
	if len(tels) != 2 {
		t.Fatalf("TEL 数量 = %d", len(tels))
	}
	// 带引号参数：引号内逗号是内容，整体为单值；重复 TYPE 参数都保留
	if len(tels[0].Params) != 3 {
		t.Fatalf("TEL 参数数 = %d", len(tels[0].Params))
	}
	if tels[0].Params[0].Name != "TYPE" || len(tels[0].Params[0].Values) != 1 ||
		tels[0].Params[0].Values[0] != "WORK,VOICE" {
		t.Errorf("TEL 引号参数 = %+v", tels[0].Params[0])
	}
	if tels[0].Params[1].Name != "TYPE" || tels[0].Params[1].Values[0] != "CELL" {
		t.Errorf("TEL 重复参数 = %+v", tels[0].Params[1])
	}
	if tels[0].Params[2].Name != "X-EXTRA" || len(tels[0].Params[2].Values) != 2 {
		t.Errorf("TEL 多值参数 = %+v", tels[0].Params[2])
	}
	if tels[0].Value != "+86-138-0000-0000" {
		t.Errorf("TEL 值 = %q", tels[0].Value)
	}
	addrs := c.Addresses()
	if len(addrs) != 1 {
		t.Fatalf("ADR 数量 = %d", len(addrs))
	}
	addr := addrs[0]
	if addr.POBox[0] != "" || addr.Extended[0] != "" {
		t.Errorf("ADR 空前分量丢失：%+v", addr)
	}
	if addr.Street[0] != "中山,路1号" || addr.Locality[0] != "北京" ||
		addr.Region[0] != "" || addr.PostalCode[0] != "100000" || addr.Country[0] != "中国" {
		t.Errorf("ADR = %+v", addr)
	}
	cats := c.Categories()
	if len(cats) != 3 || cats[0] != "朋友,同事" || cats[1] != "VIP" || cats[2] != "" {
		t.Errorf("CATEGORIES = %v（空列表项丢失）", cats)
	}
	if got := c.Note(); got != noteValue {
		t.Errorf("NOTE 折行展开错误：%q", got)
	}
	// 组名前缀与空值
	if p := c.Get("X-ABLABEL"); p == nil || p.Group != "ITEM1" || p.Value != "自定义标签" {
		t.Errorf("组名属性 = %+v", p)
	}
	if p := c.Get("X-EMPTY"); p == nil || p.Value != "" {
		t.Errorf("空值属性 = %+v", p)
	}

	// 写回后再次读取仍一致
	var buf bytes.Buffer
	if err := Write(&buf, contacts); err != nil {
		t.Fatalf("写回失败：%v", err)
	}
	checkLineLimit(t, buf.Bytes())
	contacts2, err := Parse(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("回读失败：%v", err)
	}
	c2 := contacts2[0]
	if c2.FN() != c.FN() || c2.Note() != c.Note() {
		t.Errorf("回读后 FN/NOTE 不一致")
	}
	n2 := c2.Name()
	if n2.Family[0] != "张" || n2.Given[0] != "三,丰" || n2.Additional[0] != "小;明" || n2.Suffix[0] != "博士" {
		t.Errorf("回读后 N = %+v", n2)
	}
	if a2 := c2.Addresses()[0]; a2.Street[0] != "中山,路1号" || a2.PostalCode[0] != "100000" {
		t.Errorf("回读后 ADR = %+v", a2)
	}
	if cats2 := c2.Categories(); len(cats2) != 3 || cats2[0] != "朋友,同事" || cats2[2] != "" {
		t.Errorf("回读后 CATEGORIES = %v", cats2)
	}
	if len(c2.Props) != len(c.Props) {
		t.Errorf("回读后属性数 %d != %d", len(c2.Props), len(c.Props))
	}
	for i := range c.Props {
		if c.Props[i].Name != c2.Props[i].Name || c.Props[i].Group != c2.Props[i].Group {
			t.Errorf("第 %d 个属性顺序/名称变化：%v -> %v", i, c.Props[i].Name, c2.Props[i].Name)
		}
	}
}

// TestBoundaryAndRoundTrip 覆盖：备注中出现边界字样、修改单个字段后的往返、
// 未知属性逐字节保留、75 字节折行限制。
func TestBoundaryAndRoundTrip(t *testing.T) {
	// NOTE 解码后包含独立的 BEGIN:VCARD / END:VCARD 行，不能成为边界。
	note := "第一行\nBEGIN:VCARD\nEND:VCARD\n第三行"
	unknown := `X-CUSTOM;P="a;b:c,d":值\;含\,标点\\与换行\n符`
	long := "X-LONG:" + strings.Repeat("长", 40) // 120 字节，必然折行

	input := "BEGIN:VCARD\n" + // LF 换行
		"VERSION:3.0\n" +
		"FN:测试用户\n" +
		"TEL;TYPE=WORK:111\n" +
		"TEL;TYPE=HOME:222\n" +
		"NOTE:" + escapeText(note) + "\n" +
		unknown + "\n" +
		long + "\n" +
		"END:VCARD" // 最后一行没有换行符

	contacts, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("备注中的边界字样被误识别：得到 %d 张名片", len(contacts))
	}
	c := contacts[0]
	if got := c.Note(); got != note {
		t.Errorf("NOTE = %q", got)
	}

	// 修改单个 TEL 与 NOTE
	c.Props[2].Value = "333" // TEL;TYPE=WORK
	c.SetNote("新的备注\n第二行")

	var buf bytes.Buffer
	if err := Write(&buf, contacts); err != nil {
		t.Fatalf("写回失败：%v", err)
	}
	out := buf.Bytes()
	checkLineLimit(t, out)

	// 未知属性展开后的内容逐字节不变
	lines := physicalLines(t, out)
	var unfolded []string
	for _, line := range lines {
		if strings.HasPrefix(line, " ") && len(unfolded) > 0 {
			unfolded[len(unfolded)-1] += line[1:]
		} else {
			unfolded = append(unfolded, line)
		}
	}
	foundUnknown, foundLong := false, false
	for _, line := range unfolded {
		if line == unknown {
			foundUnknown = true
		}
		if line == long {
			foundLong = true
		}
	}
	if !foundUnknown {
		t.Errorf("未知属性内容发生变化，期望 %q", unknown)
	}
	if !foundLong {
		t.Errorf("长未知属性内容发生变化")
	}

	// 回读验证：其他属性语义及顺序不变
	contacts2, err := Parse(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("回读失败：%v", err)
	}
	c2 := contacts2[0]
	if got := c2.Note(); got != "新的备注\n第二行" {
		t.Errorf("修改后 NOTE = %q", got)
	}
	tels := c2.Telephones()
	if len(tels) != 2 || tels[0].Value != "333" || tels[1].Value != "222" {
		t.Errorf("TEL = %+v", tels)
	}
	if tels[0].Params[0].Values[0] != "WORK" || tels[1].Params[0].Values[0] != "HOME" {
		t.Errorf("TEL 参数变化：%+v", tels)
	}
	if c2.FN() != "测试用户" {
		t.Errorf("FN = %q", c2.FN())
	}
	wantOrder := []string{"VERSION", "FN", "TEL", "TEL", "NOTE", "X-CUSTOM", "X-LONG"}
	if len(c2.Props) != len(wantOrder) {
		t.Fatalf("属性数 %d != %d", len(c2.Props), len(wantOrder))
	}
	for i, name := range wantOrder {
		if c2.Props[i].Name != name {
			t.Errorf("第 %d 个属性 = %s，期望 %s", i, c2.Props[i].Name, name)
		}
	}
}

// TestFoldUTF8Boundary 验证折行从不切断 UTF-8 字符。
func TestFoldUTF8Boundary(t *testing.T) {
	c := &Contact{Props: []Property{
		{Name: "NOTE", Value: strings.Repeat("汉", 50)},
	}}
	var buf bytes.Buffer
	if err := Write(&buf, []*Contact{c}); err != nil {
		t.Fatal(err)
	}
	checkLineLimit(t, buf.Bytes())
	back, err := Parse(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if back[0].Props[0].Value != strings.Repeat("汉", 50) {
		t.Errorf("折行后内容变化")
	}
}
