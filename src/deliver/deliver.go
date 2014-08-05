/**
 * User: chenggong
 * Date: 14-7-3
 * Time: 下午6:25
 */
package deliver

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

var ErrShortWrite error = errors.New("short write")

const SEND_UNIT = 7 * 188
const BUF_SIZE = SEND_UNIT * 128
const UINT64_SIZE = 8
const MAXLATENESS = 1 * 1000 * 1000 * 1000 //1s
const DELAY_CHECK_INTERVAL = 10

func Copy(src io.Reader, aux io.Reader, dst io.Writer, duration int) (written int64, err error) {
	written = int64(0)
	iobuf := make([]byte, BUF_SIZE)
	auxBuf := make([]byte, UINT64_SIZE)
	streamStartTime := int64(0)
	startTime := int64(0)
	stopDur := int64(0)
	if duration > 0 {
		stopDur = int64(duration) * 1000 * 1000 * 1000 // s -> ns
	}
	i := 0
	readed := BUF_SIZE
	var buf []byte
	for {
		i++
		if BUF_SIZE-readed < SEND_UNIT {
			nr, er := src.Read(iobuf)
			if nr > 0 {
				readed = 0
			} else {
				break
			}
			if er != nil {
				err = er
				break
			}
		}
		buf = iobuf[readed : readed+SEND_UNIT]
		readed += SEND_UNIT

		aux.Read(auxBuf)

		nw, ew := dst.Write(buf[0:SEND_UNIT])
		if nw > 0 {
			written += int64(nw)
		}
		if ew != nil {
			err = ew
			//				break
		}

		if i%DELAY_CHECK_INTERVAL == 0 {
			streamTime := int64(binary.BigEndian.Uint64(auxBuf))
			now := time.Now().UnixNano()
			if startTime == 0 {
				startTime = now
				streamStartTime = streamTime
				fmt.Printf("stream time start with:%d\n", streamStartTime)
			}
			wallDur := now - startTime
			streamDur := (streamTime - streamStartTime) * 1000 / 27 // 27MHz -> ns
			if stopDur > 0 && streamDur > stopDur {
				break
			}
			delay := streamDur - wallDur
			if delay > 0 {
				time.Sleep(time.Duration(delay))
			} else if delay < -MAXLATENESS {
				fmt.Println("too much lateness, reset clock!")
				startTime = now
				streamStartTime = streamTime
			}
		}
	}
	return written, err
}
