package textsplit

import "strings"

type Splitter struct {
	ChunkSize int
	Overlap   int
	Separator string // '\n' by default
}

func NewTextSplitter(chunkSize int, overlap int, separator string) *Splitter {
	// 防御性编程：重叠部分不能大于等于切片总大小，否则会导致死循环
	if overlap >= chunkSize {
		overlap = chunkSize - 1
	}
	if chunkSize <= 0 {
		chunkSize = 500 // 兜底默认值
	}

	return &Splitter{
		ChunkSize: chunkSize,
		Overlap:   overlap,
		Separator: separator,
	}
}

func (s *Splitter) Split(text string) []string {
	if text == "" {
		return nil
	}

	// 1. 将底层字符转化为unicode字符切片，解决中文乱码
	runes := []rune(text)
	textLength := len(runes)

	var chunks []string
	start := 0
	// 2. 按照指定的chunkSize和overlap进行切分
	for start < textLength {
		end := start + s.ChunkSize
		if end > textLength {
			end = textLength
		}

		chunk := string(runes[start:end])

		chunk = strings.TrimSpace(chunk) // 去除首尾空白字符

		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		start += s.ChunkSize - s.Overlap
	}

	return chunks
}
