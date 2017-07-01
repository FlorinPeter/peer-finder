// A small utility program to lookup hostnames of endpoints in a service.
package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"io"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	pollPeriod = 1 * time.Second
)

var (
	svc       = flag.String("service", "", "Governing service responsible for the DNS records of the domain this pod is in.")
	namespace = flag.String("ns", "", "The namespace this pod is running in. If unspecified, the POD_NAMESPACE env var is used.")
	domain    = flag.String("domain", "cluster.local", "The Cluster Domain which is used by the Cluster.")
)

func lookup(svcName string) (sets.String, error) {
	endpoints := sets.NewString()

	addrs, err := net.LookupHost(svcName)
	log.Printf("LookupHost %v %v", addrs, err)

	for _, addr := range addrs {
		endpoints.Insert(addr)
	}

	return endpoints, nil
}

func WriteStringToFile(filepath, s string) error {
	fo, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fo.Close()

	_, err = io.Copy(fo, strings.NewReader(s))
	if err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	ns := *namespace
	if ns == "" {
		ns = os.Getenv("POD_NAMESPACE")
	}
	if *svc == "" || ns == "" {
		log.Fatalf("Incomplete args, require -service and -ns or an env var for POD_NAMESPACE.")
	}

	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		log.Fatalf("Failed to get hostname env")
	}

	log.Printf("hostname %v", hostname)

	svcLocalSuffix := strings.Join([]string{"endpoints", *domain}, ".")
	myName := strings.Join([]string{hostname, *svc, ns, svcLocalSuffix}, ".")

	log.Printf("myName %v", myName)

	addrs, err := net.LookupHost(hostname)
	log.Printf("addrs %v %v", addrs, err)

	if len(addrs) == 0 {
		log.Fatalf("ip not found")
	}

	myIP := addrs[0]

	query := strings.Join([]string{*svc, ns, svcLocalSuffix}, ".")

	peers, err := lookup(query)
	if err != nil {
		log.Fatalf("%v", err)
	}

	peerList := ""
	for _, peer := range peers.List() {
		log.Printf("peer %v", peer)
		if peer != myIP {
			peerList = strings.Join([]string{peerList, peer}, " ")
		}
	}

	peerList = strings.Join([]string{peerList, hostname}, " ")

	log.Printf("peerList %v", peerList)

	WriteStringToFile("/tmp/peers", peerList)

	log.Printf("Peer finder exiting")
}
