package bytecount

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

// func ConvertBytes(b float64) string {
// 	if b < 1 {
// 		return fmt.Sprintf("%.01f B", b)
// 	}
// 	units := []string{"B", "KB", "MB", "GB", "TB"}
// 	exp := math.Min(float64(len(units)-1), math.Floor(math.Log2(b)/10))
// 	value := b / math.Pow(1024, exp)
// 	return fmt.Sprintf("%.01f %s", value, units[int(exp)])
// }

// func (bc *ByteCount) getUnits() []string {
// 	return []string{"B", "KB", "MB", "GB", "TB"}
// }

// func (bc *ByteCount) Convert() {
// 	units := bc.getUnits()
// 	i := 0
// 	for bc.Total >= 1024 && i < len(units)-1 {
// 		bc.Total /= 1024
// 		i++
// 	}
// 	bc.Unit = units[i]
// }

// func (bc *ByteCount) Add(b int) {
// 	units := []string{"B", "KB", "MB", "GB", "TB"}
// 	i := 0
// 	c := b
// 	for c >= 1024 && i < len(units)-1 {
// 		c /= 1024
// 		i++
// 	}
// 	bc.Unit = units[i]
// 	bc.Total = float64(c)
// }

// func (bc *ByteCount) CalcSpeed(elapsedTime int) int {
// 	return bc.Total / elapsedTime
// }
