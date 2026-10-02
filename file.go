package vcard

import "os"

// ReadFile 从本地文件读取并解析名片。
func ReadFile(path string) ([]*Card, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// WriteFile 将名片序列化后写入本地文件。
func WriteFile(path string, cards []*Card) error {
	return os.WriteFile(path, Marshal(cards), 0o644)
}
