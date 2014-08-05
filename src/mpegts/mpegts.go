package mpegts

import (
	"github.com/ziutek/dvb/ts"
	"io"
	"log"
	"time"
)

func rb16(r []byte) uint16 {
	return uint16(r[0])<<8 + uint16(r[1])
}

func ri16(r []byte) uint16 {
	return uint16(r[0])<<8 + uint16(r[1])
}

func ri8(r []byte) int {
	return int(int8(r[0]))
}

func rb8(r []byte) int {
	return int(r[0])
}

func parsePesDts(r []byte) int64 {
	return (int64(r[0])&0x0e)<<29 |
		int64(rb16(r[1:])>>1)<<15 |
		int64(rb16(r[3:]))>>1
}

const (
	V_H264 = 1
	V_MP2  = 2
	A_MP2  = 3
	A_AAC  = 4
	A_MP3  = 5
	A_MP4  = 6
	A_AC3  = 7
)

const MAX_INTERVAL time.Duration = 500 * 1000 * 1000 * 1000 //500ms
const POW2_33 time.Duration = 8589934592 * 300

var STREAM_TYPE map[int]int = map[int]int{
	0x02: V_MP2,
	0x03: A_MP3,
	0x04: A_MP2,
	0x0f: A_AAC,
	0x11: A_MP4,
	0x1b: V_H264,
	0x81: A_AC3,
}

type tsPacket struct {
	codec int
	data  []byte
}

type tsFilter struct {
	pos, size    int
	codec        int
	header, data []byte
}

type tsStream struct {
	tsmap map[uint16]*tsFilter
}

const TS_PKT_SIZE = 188

func Read(reader io.Reader) (err error) {
	pr := ts.NewPktStreamReader(reader)
	pkt := new(ts.ArrayPkt)
	done := false
	processed := int64(0)
	lastPcrPos := int64(0)
	tsMap := map[uint16]*tsFilter{}
	var pmtPid uint16 = 0
	var pcrPid uint16 = 0

	var firstPcr time.Duration = 0
	var lastPcrDelta time.Duration = 0
	var lastPcr time.Duration = 0
	var pcrDelta time.Duration = 0
	var duration time.Duration = 0
	pcr_cnt := 0

	for !done {
		err := pr.ReadPkt(pkt)
		processed += 1
		if err != nil {
			if err == io.EOF {
				log.Printf("read finished in offset %d", (processed-1)*188)
				done = true
			} else if err == ts.ErrSync {
				log.Printf("err %s in offset %d", err, processed)
				continue
			} else {
				log.Printf("err %s in offset %d", err, processed)
				break
			}
		}
		pid := pkt.Pid()

		tss := tsMap[pid]
		if tss == nil {
			tss = &tsFilter{}
			tsMap[pid] = tss
			log.Printf(" new tss [%d]", pid)
		}

		isStart := pkt.PayloadStart()
		hasAdapt := pkt.ContainsAF()

		if hasAdapt {
			af := pkt.AF()
			if af.Flags().ContainsPCR() {
				pcr, err := af.PCR()
				if err != nil {
					log.Println(err)
				}
				currentPcr := pcr.Nanosec()
				if firstPcr == time.Duration(0) {
					firstPcr = currentPcr
				}
				if pcrPid != 0 && pcrPid != pid {
					log.Println("#pcr pid changed:", pcrPid)
				}
				pcrPid = pid //pcr pid found
				if pcr_cnt > 0 {
					pcrDelta = currentPcr - lastPcr
					if pcrDelta < 0 || pcrDelta > MAX_INTERVAL {
						/* Do not change the slope - consider CBR */
						log.Println("!!!!!!!!PCR discontinuity", lastPcr, currentPcr, processed)
						pcrDelta = lastPcrDelta
					} else {
						lastPcrDelta = pcrDelta
					}
					duration += pcrDelta
				}
				//				log.Println("pcr interval:", pcrDelta, time.Duration(int64(pcrDelta)/(processed - lastPcrPos)))
				if pcrDelta > 0 {
					log.Println("bitrate:", int64(pcrDelta)/((processed-lastPcrPos)*188), (processed - lastPcrPos), pcrDelta, (currentPcr - firstPcr))
				}
				lastPcr = currentPcr
				pcr_cnt++
				lastPcrPos = processed
			}
		}

		if !pkt.ContainsPayload() {
			continue
		}

		p := pkt.Payload()

		parsePAT := func() {
			p := tss.data
			if len(p) < 8 {
				return
			}
			tid := rb8(p)
			p = p[8:]
			if tid != 0x0 {
				return
			}
			for len(p) >= 4 {
				sid := ri16(p)
				pmtPid = rb16(p[2:]) & 0x1fff
				if sid > 0 { //pmt pid found
					//					log.Printf("  pat: sid 0x%x pmt pid %d", sid, pmtPid)
					break
				}
				p = p[4:]
			}
			return
		}

		parsePMT := func() {
			//			log.Printf("=====%d pmt=====", processed)
			p := tss.data
			if len(p) < 8 {
				return
			}
			tid := rb8(p) // table id , must be 0x02
			p = p[8:]
			if tid != 0x2 {
				log.Printf("  pmt: table id is wrong: 0x%x", tid)
				return
			}
			if len(p) < 4 {
				return
			}
			pcrPid = rb16(p) & 0x1fff
			//			log.Println("pcr pid found in PMT is :", pcrPid)
			p = p[4:]
			for len(p) >= 5 {
				strType := rb8(p)
				strPid := rb16(p[1:]) & 0x1fff
				descLen := rb16(p[3:]) & 0xfff
				t := tsMap[strPid]
				if t == nil {
					t = &tsFilter{}
					tsMap[strPid] = t
				}
				//				log.Println("---------type pid is :", strType, strPid, STREAM_TYPE[strType])
				if STREAM_TYPE[strType] > 0 {
					t.codec = STREAM_TYPE[strType]
				} else {
					log.Printf("unspported stream type 0x%x %d", strType, strPid)
				}
				p = p[5+descLen:]
			}
		}

		parseSection2 := func(p []byte) {
			tss.data = append(tss.data, p...) //append section
			if tss.size <= 0 {                //unknown section size
				if len(tss.data) >= 3 {
					tss.size = int(rb16(tss.data[1:3])&0xfff + 3)
					if tss.size > 4096 {
						tss.size = -1
						return
					}
				}
			}
			if len(tss.data) >= tss.size {
				tss.data = tss.data[0:tss.size]
				if pid == 0x00 { //is PAT
					parsePAT()
					//					log.Printf("%d pat, pmt id is %d\n", processed, pmtPid)
				}
				if pmtPid > 0 && pid == pmtPid { //is PMT
					parsePMT()
				}
				tss.data = []byte{}
				tss.size = 0
			}
		}

		parseSection := func() {
			if isStart {
				if len(p) < 1 { //has no payload
					return
				}
				sz := rb8(p)
				p = p[1:]
				if sz > len(p) {
					return
				}
				parseSection2(p)
			} else {
				parseSection2(p)
			}
		}

		//		parsePes2 := func() {
		//			tss.data = tss.data[:tss.pos]
		//		}

		parsePes := func() {
			if isStart {
				//				if tss.codec == V_H264 {
				//					log.Println("H264 Video HERE++++%d", processed)
				//				} else if tss.codec == V_MP2 {
				//					log.Println("MP2 Video HERE ++++%d", processed)
				//				} else if tss.codec == A_MP3 {
				//					log.Println("MP3 Audio HERE ++++%d", processed)
				//				}
				if len(tss.data) > 0 {
					//	parsePes2()
				}
				tss.header = make([]byte, 9)
				copy(tss.header, p[0:9])
				headerSize := int(tss.header[8])
				//				if headerSize > 0 {
				//					pesHeader := p[9 : 9+headerSize]
				//					dts := parsePesDts(pesHeader)
				//				}
				totalSize := 200000

				tss.data = make([]byte, totalSize)
				tss.pos = 0
				p = p[9+headerSize:]
			}

			if len(tss.data) > 0 {
				l := tss.pos + len(p)
				if l > len(tss.data) {
					l = len(tss.data)
				}
				//				copy(tss.data[tss.pos:l], p)
				tss.pos += len(p)
			}

			if tss.pos >= len(tss.data) {
				//parsePes2()
			}
		}

		if pid == 0x00 || (pmtPid > 0 && pid == pmtPid) { //is PAT or PMT
			parseSection()
		} else { //is PES
			parsePes()
		}

	}
	return
}
