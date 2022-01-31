package main

import (
	"crypto/tls"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	proxy "github.com/esote/http-proxy"
	"github.com/esote/openshim2"
)

func init() {
	if err := openshim2.LazySysctls(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	const (
		cert = "/etc/ssl/esote.net.fullchain.pem"
		key  = "/etc/ssl/private/esote.net.key"
	)

	if err := openshim2.Unveil(cert, "r"); err != nil {
		log.Fatal(err)
	}

	if err := openshim2.Unveil(key, "r"); err != nil {
		log.Fatal(err)
	}

	if err := openshim2.Pledge("stdio rpath inet", ""); err != nil {
		log.Fatal(err)
	}

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.X25519,
		},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	stop1 := make(chan bool)
	stop2 := make(chan bool)

	proxies := &proxy.Proxies{
		Proxies: []proxy.ReverseProxy{
			{
				Port:        ":80",
				Stop:        stop1,
				StopTimeout: 10 * time.Second,
			},
			{
				Cert:        cert,
				Key:         key,
				Port:        ":443",
				TLSConfig:   cfg,
				Stop:        stop2,
				StopTimeout: 10 * time.Second,
			},
		},
	}

	routes := map[string]string{
		"":      "8442",
		"fmtc.": "8443",
		"chat.": "8444",
		"git.":  "8445",
	}

	const domain = "esote.net/"

	for key, value := range routes {
		proxies.Proxies[0].Routes = append(proxies.Proxies[0].Routes, proxy.Route{
			From: key + domain,
			To:   "http://" + key + "localhost:8441",
		})
		proxies.Proxies[1].Routes = append(proxies.Proxies[1].Routes, proxy.Route{
			From: key + domain,
			To:   "http://localhost:" + value,
		})
	}

	errs := proxy.Proxy(proxies)

	var mu sync.Mutex
	fan := make(chan bool)
	sleep := make(chan bool, 1)
	running := true

	go func() {
		for {
			v := <-fan
			stop1 <- v
			stop2 <- v
			sleep <- v
		}
	}()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGUSR1)

		for {
			select {
			case <-sigs:
				mu.Lock()
				fan <- running
				if !running {
					errs = proxy.Proxy(proxies)
				}
				running = !running
				mu.Unlock()
			}
		}
	}()

	for {
	inner:
		for {
			select {
			case err, ok := <-errs:
				if ok {
					log.Println(err)
				} else {
					mu.Lock()
					if running {
						log.Fatal("all servers died")
					}
					mu.Unlock()
					break inner
				}
			}
		}

		<-sleep
	}
}
