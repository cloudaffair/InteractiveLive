package es

import(
	"github.com/elastic/go-elasticsearch"
	"net/http"
	"time"
	"net"
	"crypto/tls"
)

func GetClient(elasticServers []string) (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: elasticServers,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second,
			DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS11,
			},
		},
	}

	return elasticsearch.NewClient(cfg)
}


