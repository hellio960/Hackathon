package unit

import (
	"math"
	"strconv"
)

// Byte 字节
type Byte int64

var (
	// B 字节
	B Byte = 1
	// KB 千字节
	KB = Byte(1024) * B
	// MB 兆字节
	MB = Byte(1024) * KB
	// GB 吉字节
	GB = Byte(1024) * MB
	// TB 太字节
	TB = Byte(1024) * GB
)

func ByteConvert(size Byte) string {
	var base float64
	var suffixes = [5]string{"B", "K", "M", "G", "T"}
	if base = math.Log(float64(size)) / math.Log(1024); base < 0 || size <= 0 || math.Floor(base) > 4 || math.IsInf(base, 0) {
		return "0 B"
	}
	getSize := Round(math.Pow(1024, base-math.Floor(base)), .5, 2)
	getSuffix := suffixes[int(math.Floor(base))]
	return strconv.FormatFloat(getSize, 'f', -1, 64) + " " + string(getSuffix)
}

func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}
