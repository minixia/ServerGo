package tsindex

import (
	"encoding/json"
	"fmt"
	"github.com/ziutek/dvb/ts"
	"io"
	"time"
)

const MAX_INTERVAL time.Duration = 100 * 1000 * 1000 * 1000 //100ms

type IndexRecord struct {
	Dur      time.Duration `json:"d"`
	Pcr      time.Duration `json:"p"`
	Offset   uint64        `json:"o"`
	Keyframe bool          `json:"k"`
}

func (ir *IndexRecord) toJSON() (buf []byte) {
	buf, _ = json.Marshal(ir)
	return
}

func Index(reader io.Reader, writer io.Writer) {
	pr := ts.NewPktStreamReader(reader)
	pkt := new(ts.ArrayPkt)
	pcr_cnt := 0
	var lastPcrDelta time.Duration = 0
	var lastPcr time.Duration = 0
	var pcrDelta time.Duration = 0
	var duration time.Duration = 0
	done := false
	offset := uint64(0)
	for !done {
		err := pr.ReadPkt(pkt)
		offset += uint64(len(pkt.Bytes()))
		if err != nil {
			if err == io.EOF || err == ts.ErrSync {
				fmt.Printf("read %s in %d!\n", err, offset)
				done = true
			} else {
				fmt.Printf("error in %d is %s\n", offset, err)
				break
			}
		}

		if !pkt.Flags().ContainsAF() {
			continue
		}
		af := pkt.AF()
		if !af.Flags().ContainsPCR() {
			continue
		}
		pcr, err := af.PCR()
		if err != nil {
			fmt.Println(err)
			continue
		}
		if pcr_cnt > 0 {
			pcrDelta = pcr.Nanosec() - lastPcr
			if pcrDelta < 0 || pcrDelta > MAX_INTERVAL {
				pcrDelta = lastPcrDelta
			}
			lastPcrDelta = pcrDelta
			duration += pcrDelta
		}

		record := IndexRecord{
			Dur:    duration,
			Pcr:    pcr.Nanosec(),
			Offset: offset - uint64(len(pkt.Bytes())),
		}
		writer.Write(record.toJSON())

		lastPcr = pcr.Nanosec()
		pcr_cnt++
	}
	return
}
