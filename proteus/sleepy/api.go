package main

import "fmt"

type BuildData struct {
	Secure   bool
	Host     string
	Port     int
	Username string
	Password string
}

func (bd BuildData) ToPrefix() string {
	proto := "http"
	if bd.Secure {
		proto = "https"
	}
	port := bd.Port
	if port == 0 {
		if bd.Secure {
			port = 443
		} else {
			port = 80
		}
	}
	return fmt.Sprintf("%s://%s:%d", proto, bd.Host, port)
}

const (
	HEADER      = "Header"
	STATUS_CODE = "Status"
	BODY        = "Body"
)
