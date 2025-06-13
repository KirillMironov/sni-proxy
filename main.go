package main

import (
	"encoding/base64"
	"log"
	"net/http"
	"net/url"

	"github.com/elazarl/goproxy"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ListenAddress string `envconfig:"LISTEN_ADDRESS" default:":8080"`
	UpstreamProxy string `envconfig:"UPSTREAM_PROXY" required:"true"`
	ProxyUsername string `envconfig:"PROXY_USERNAME" required:"true"`
	ProxyPassword string `envconfig:"PROXY_PASSWORD" required:"true"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return err
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.KeepDestinationHeaders = true
	proxy.AllowHTTP2 = true

	upstream, err := url.Parse(config.UpstreamProxy)
	if err != nil {
		return err
	}

	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(config.ProxyUsername+":"+config.ProxyPassword))

	proxy.Tr = &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			req.Header.Set("Proxy-Authorization", auth)
			return upstream, nil
		},
	}

	log.Println("adapter is listening on ", config.ListenAddress)

	return http.ListenAndServe(config.ListenAddress, proxy)
}
