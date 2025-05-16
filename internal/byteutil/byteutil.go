package byteutil

type Byte struct {
	Unit  string
	Total float64
}

func ConvertBytes(b float64) *Byte {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	for b >= 1024 && i < len(units)-1 {
		b /= 1024
		i++
	}
	return &Byte{
		Unit:  units[i],
		Total: b,
	}
}

func Split(b []byte, n int) [][]byte {
	if n <= 0 {
		return nil
	}
	result := make([][]byte, 0, n)
	chunkSize := len(b) / n
	remainder := len(b) % n
	start := 0
	for i := 0; i < n; i++ {
		end := start + chunkSize
		if i < remainder {
			end++
		}
		if end > len(b) {
			end = len(b)
		}
		result = append(result, b[start:end])
		start = end
	}
	return result
}
