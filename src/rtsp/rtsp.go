package rtsp

import (
	"errors"
	"io"
	"net/textproto"
	"regexp"
	"strconv"
)

/**
this is a rtsp protocol implmenets, follow rfc2326[http://www.ietf.org/rfc/rfc2326.txt]
*/

/**
RFC2326-6.1 Request Line

  Request-Line = Method SP Request-URI SP RTSP-Version CRLF

   Method         =         "DESCRIBE"              ; Section 10.2
                  |         "ANNOUNCE"              ; Section 10.3
                  |         "GET_PARAMETER"         ; Section 10.8
                  |         "OPTIONS"               ; Section 10.1
                  |         "PAUSE"                 ; Section 10.6
                  |         "PLAY"                  ; Section 10.5
                  |         "RECORD"                ; Section 10.11
                  |         "REDIRECT"              ; Section 10.10
                  |         "SETUP"                 ; Section 10.4
                  |         "SET_PARAMETER"         ; Section 10.9
                  |         "TEARDOWN"              ; Section 10.7
                  |         extension-method

  extension-method = token

  Request-URI = "*" | absolute_URI

  RTSP-Version = "RTSP" "/" 1*DIGIT "." 1*DIGIT
*/
const METHODS = map[string]string{
	"DESCRIBE":      "describe",
	"SETUP":         "setup",
	"PLAY":          "play",
	"PAUSE":         "pause",
	"TEARDOWN":      "teardown",
	"ANNOUNCE":      "announce",
	"GET_PARAMETER": "get parameter",
	"SET_PARAMETER": "set parameter",
	"OPTIONS":       "options",
}

/**
  Status-Code  =     "100"      ; Continue
               |     "200"      ; OK
               |     "201"      ; Created
               |     "250"      ; Low on Storage Space
               |     "300"      ; Multiple Choices
               |     "301"      ; Moved Permanently
               |     "302"      ; Moved Temporarily
               |     "303"      ; See Other
               |     "304"      ; Not Modified
               |     "305"      ; Use Proxy
               |     "400"      ; Bad Request
               |     "401"      ; Unauthorized
               |     "402"      ; Payment Required
               |     "403"      ; Forbidden
               |     "404"      ; Not Found
               |     "405"      ; Method Not Allowed
               |     "406"      ; Not Acceptable
               |     "407"      ; Proxy Authentication Required
               |     "408"      ; Request Time-out
               |     "410"      ; Gone
               |     "411"      ; Length Required
               |     "412"      ; Precondition Failed
               |     "413"      ; Request Entity Too Large
               |     "414"      ; Request-URI Too Large
               |     "415"      ; Unsupported Media Type
               |     "451"      ; Parameter Not Understood
               |     "452"      ; Conference Not Found
               |     "453"      ; Not Enough Bandwidth
               |     "454"      ; Session Not Found
               |     "455"      ; Method Not Valid in This State
               |     "456"      ; Header Field Not Valid for Resource
               |     "457"      ; Invalid Range
               |     "458"      ; Parameter Is Read-Only
               |     "459"      ; Aggregate operation not allowed
               |     "460"      ; Only aggregate operation allowed
               |     "461"      ; Unsupported transport
               |     "462"      ; Destination unreachable
               |     "500"      ; Internal Server Error
               |     "501"      ; Not Implemented
               |     "502"      ; Bad Gateway
               |     "503"      ; Service Unavailable
               |     "504"      ; Gateway Time-out
               |     "505"      ; RTSP Version not supported
               |     "551"      ; Option not supported
               |     extension-code
*/
const RSP_STATUS = map[string]string{
	"200": "OK",
	"302": "Moved Temporarily",
	"400": "Bad Request",
	"403": "Forbidden",
	"404": "Not Found",
	"415": "Unsupported Media Type",
	"454": "Session Not Found",
	"457": "Invalid Range",
	"462": "Destination unreachable",
	"500": "Internal Server Error",
	"501": "Not Implemented",
	"502": "Bad Gateway",
}

type Request struct {
	method string
	uri    string
	header MIMEHeader
	cseq   int
}

func ProcessRequest(reader io.Reader) (request Request, err error) {
	tr := textproto.NewReader(&reader)
	firstLine, er := tr.ReadLine()
	if er != nil {
		err = er
		return
	}
	request.cseq = 1
	request.method, request.uri, er = parseMethod(firstLine)
	if er != nil {
		err = er
		return
	}
	request.header, er = tr.ReadMIMEHeader()
	if er != nil {
		err = er
	}
	if request.header.Get("CSeq") == "" {
		err = "CSeq not found"
		return
	}
	request.cseq = request.header.Get("CSeq")
	return
}

func parseMethod(line string) (method string, uri string, err error) {
	re, _ := regexp.Compile("[^\\s+]+")
	re.FindAllString(line, -1)
	method = line[0]
	if METHODS[method] != nil {
		uri = line[1]
	} else {
		err = errors.New("[RTSP]Unspported method:" + method)
	}
	return
}

func Response(writer io.Writer, cseq int, status string, header map[string]string, body string) (err error) {
	resp := "RTSP/1.0 " + status + " " + RSP_STATUS[status] + "\r\n"
	resp += "CSeq:" + cseq + "\r\n"
	if header != nil {
		for key, value := range header {
			resp += key + ":" + value + "\r\n"
		}
	}
	if body != nil && len(body) > 0 {
		resp += "Content-Length:" + strconv.Itoa(len(body))
	}
	resp += "\r\n"
	err = writer.Write([]byte(resp))
	return
}
