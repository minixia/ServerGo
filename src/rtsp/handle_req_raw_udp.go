package rtsp

func Handle(req Request) (status string, header map[string]string, body string, err error) {
	req.header.Get()
}
