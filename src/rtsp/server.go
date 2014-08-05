package rtsp

import (
	"bufio"
	"net"
)

type Handler interface {
	Handle(req Request) (status string, header map[string]string, body string, err error)
}

func Listen(laddr string, h Handler) (err error) {
	for {
		listener, er := net.Listen("tcp", laddr)
		if er != nil {
			err = er
			return
		}
		conn, er := listener.Accept()
		if er != nil {
			err = er
			return
		}
		go process(conn, h)
	}
}

func process(conn net.Conn, h Handler) {
	bReader := bufio.NewReader(conn)
	request, err := ProcessRequest(bReader)
	if err != nil {
		Response(conn, request.cseq, "400", nil, err.Error())
	}
	status, header, body, err := h.Handle(request)
	if err == nil {
		Response(conn, request.cseq, status, header, body)
	} else {
		Response(conn, request.cseq, status, header, err.Error())
	}
}
