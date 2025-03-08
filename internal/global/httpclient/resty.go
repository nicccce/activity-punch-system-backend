package httpclient

import (
	"github.com/go-resty/resty/v2"
	"time"
)

var Client *resty.Client

func Init() {
	Client = resty.New().SetTimeout(10 * time.Second)
}
