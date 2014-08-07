/**
 * User: chenggong
 * Date: 14-7-3
 * Time: 下午7:27
 */
package main

import (
	"bufio"
	"deliver"
	"flag"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
)

const BUF_SIZE = 7 * 188 * 256

var logger = log.New(os.Stdout, "", log.Ldate|log.Lmicroseconds|log.Lshortfile)

func play(file string, target string, dur int) {
	conn, err := net.Dial("udp", target)
	if err != nil {
		logger.Println(err)
		return
	}
	defer conn.Close()
	tsFile, err := os.Open(file + ".ts")
	defer tsFile.Close()
	if err != nil {
		logger.Println(err)
		return
	}
	auxFile, err := os.Open(file + ".aux")
	defer auxFile.Close()
	if err != nil {
		logger.Println(err)
		return
	}
	tsReader := bufio.NewReaderSize(tsFile, BUF_SIZE)
	auxReader := bufio.NewReaderSize(auxFile, BUF_SIZE)
	log.Printf("send stream %s.ts to target %s", file, target)
	n, err := deliver.Copy(tsReader, auxReader, conn, dur)

	if err != nil {
		logger.Println(err)
	}
	logger.Printf("send %d done!\n", n)
}

func forward(src string, dst string) {
	addr, _ := net.ResolveUDPAddr("udp", src)
	var srcConn, _ = net.ListenUDP("udp", addr)
	defer srcConn.Close()
	dstConn, _ := net.Dial("udp", dst)
	defer dstConn.Close()
	buf := make([]byte, 1316)
	for {
		n, _, _ := srcConn.ReadFromUDP(buf)
		if n > 0 {
			dstConn.Write(buf)
		} else if n < 0 {
			break
		}
	}
}

func main() {
	source := flag.String("s", "", "stream from udp source")
	file := flag.String("f", "", "ts file to delivery")
	dst := flag.String("o", "127.0.0.1", "output udp to")
	threadNum := flag.Int("t", 1, "thread number")
	dur := flag.Int("d", 0, "max play duration")
	concurrent := flag.Int("c", 1, "concurrent ")
	monitor := flag.String("m", "127.0.0.1:3000", "monitor stream ip&port(udp)")
	flag.Parse()
	runtime.GOMAXPROCS(*threadNum)
	if *source != "" {
		forward(*source, *dst)
	} else {
		if *concurrent > 1 {
			for i := 0; i < *concurrent; i++ {
				*dst = *dst + ":" + strconv.Itoa(10000 + i)
				go play(*file, *dst, *dur)
			}
		}
		play(*file, *monitor, *dur)
	}

}
