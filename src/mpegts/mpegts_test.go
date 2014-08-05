package mpegts

import (
	"bufio"
	"os"
	"testing"
)

func Test_Read(t *testing.T) {
	//	tsFile, _ := os.Open("/Users/chenggong/Share/ts_files/h264_bad_head.ts")
	//	tsFile, _ := os.Open("/Users/chenggong/Share/ts_files/jstv.ts") //vbr
	tsFile, _ := os.Open("/Users/chenggong/Desktop/ma.ts")
	defer tsFile.Close()
	Read(bufio.NewReaderSize(tsFile, 7*188*256))
}
