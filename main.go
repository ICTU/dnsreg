package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

var (
	socketPath  = os.Getenv("SOCKET_PATH")
	etcdClient  = createEtcdClient(os.Getenv("ETCD_URL"))
	etcdBaseKey = os.Getenv("ETCD_BASE_KEY")
)

func parseData(data, sep string) (string, string) {
	items := strings.Split(data, sep)
	if len(items) == 1 {
		return "", ""
	} else {
		return strings.TrimSpace(items[0]), strings.TrimSpace(items[1])
	}
}

func reverseArray(a []string) []string {
	for left, right := 0, len(a)-1; left < right; left, right = left+1, right-1 {
		a[left], a[right] = a[right], a[left]
	}
	return a
}

func fqdnToEtcdKey(etcdBaseKey, fqdn string) string {
	etcdKeys := reverseArray(strings.Split(fqdn, "."))
	etcdKeys = append([]string{etcdBaseKey}, etcdKeys...)
	return filepath.Join(etcdKeys...)
}

func register(etcdClient client.KeysAPI, etcdBaseKey, fqdn string, ip string) {

	etcdKey := fqdnToEtcdKey(etcdBaseKey, fqdn)
	_, err := etcdClient.Set(context.Background(), etcdKey, fmt.Sprintf(`{"host": "%s"}`, ip), nil)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Created DNS record for '%s', at key '%s', with ip '%s'", fqdn, etcdKey, ip)
	}
}

func receiveData(c net.Conn) {
	for {
		buf := make([]byte, 512)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]
		log.Printf("Received: %s", string(data))
		fqdn, ip := parseData(string(data), "|")
		if govalidator.IsDNSName(fqdn) && govalidator.IsIP(ip) {
			register(etcdClient, etcdBaseKey, fqdn, ip)
		} else {
			log.Println("Received data not valid.")
		}

	}
}

func createEtcdClient(etcdBaseUrl string) client.KeysAPI {
	cfg := client.Config{
		Endpoints:               []string{etcdBaseUrl},
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	return client.NewKeysAPI(c)
}

func main() {
	log.Println("Starting dnsreg server")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to listen on unix socket: %s", err)
	}

	// unlink the socket when shutting down
	// http://stackoverflow.com/questions/16681944/how-to-reliably-unlink-a-unix-domain-socket-in-go-programming-language
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func(ln net.Listener, c chan os.Signal) {
		sig := <-c
		log.Printf("Caught signal %s: shutting down.", sig)
		ln.Close()
		os.Exit(0)
	}(ln, sigc)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go receiveData(conn)
	}
}
