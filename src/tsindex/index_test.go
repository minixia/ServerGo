package tsindex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

type Arr struct {
	records []IndexRecord
}

func (arr *Arr) Write(p []byte) (n int, err error) {
	var record IndexRecord
	json.Unmarshal(p, &record)
	arr.records = append(arr.records, record)
	return len(p), nil
}

func Test_Index(t *testing.T) {
	tsFile, _ := os.Open("/Users/chenggong/Share/ts_files/normal1.ts")
	defer tsFile.Close()
	var arr Arr
	Index(bufio.NewReaderSize(tsFile, 7*188*256), &arr)
	dur := arr.records[len(arr.records)-1].Dur
	fmt.Printf("records num: %d\n", len(arr.records))
	fmt.Printf("delta offset between to pcr is %d\n", arr.records[1].Offset-arr.records[0].Offset)
	fmt.Printf("dur is %02d:%02d:%02d.%d \n", dur/time.Hour, (dur%time.Hour)/time.Minute, (dur%time.Minute)/time.Second, (dur%time.Second)/time.Microsecond)
}
