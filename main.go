package main

import (
  "fmt"
  "log"
  "time"
  "strings"
  "os"
  "syscall"
  "io"
  "bytes"
  "golang.org/x/net/context"
  "github.com/coreos/etcd/client"
  )

func parseData(data, sep string) (string, string) {
  items := strings.Split(data, sep)
  return strings.TrimSpace(items[0]), strings.TrimSpace(items[1])
}

func reverseArray(a []string) []string {
  for left, right := 0, len(a)-1; left < right; left, right = left+1, right-1 {
  	a[left], a[right] = a[right], a[left]
  }
  return a
}

func fqdnToEtcdKey(etcdBaseKey, fqdn string) string {
  baseKey := strings.TrimRight(etcdBaseKey, "/")
  etcdKeys := reverseArray(strings.Split(fqdn, "."))
  return baseKey + "/" + strings.Join(etcdKeys, "/")
}

func register(etcdClient client.KeysAPI, etcdBaseKey, data string) {
  fqdn, ip := parseData(data, "|")
  etcdKey := fqdnToEtcdKey(etcdBaseKey, fqdn)
  _, err := etcdClient.Set(context.Background(), etcdKey, fmt.Sprintf(`{"host": "%s"}`, ip), nil)
  if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Created DNS record for '%s', at key '%s', with ip '%s'", fqdn, etcdKey, ip)
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
  pipeName := os.Getenv("PIPE_PATH")
  etcdClient := createEtcdClient(os.Getenv("ETCD_URL"))
  etcdBaseKey := os.Getenv("ETCD_BASE_KEY")

  syscall.Mkfifo(pipeName, 0666)
  for {
    pipe, err := os.OpenFile(pipeName, os.O_RDONLY, os.ModeNamedPipe)
    if err != nil {
  		log.Fatal(err)
  	}

    var buff bytes.Buffer
    io.Copy(&buff, pipe)
    pipe.Close()
    go register(etcdClient, etcdBaseKey, buff.String())
  }
}
