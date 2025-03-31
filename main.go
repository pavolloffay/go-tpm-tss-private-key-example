package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"slices"

	keyfile "github.com/foxboron/go-tpm-keyfiles"
	"github.com/google/go-tpm-tools/simulator"
	"github.com/google/go-tpm/tpm2/transport"
	"github.com/google/go-tpm/tpmutil"
)

const (
	clientKey  = "edge-cert.key"
	clientCert = "edge-cert.crt"
	caCert     = "ca.pem"

	serverEndpoint = "localhost:9443"
)

func main() {
	tpm, err := OpenTPM("/dev/tpmrm0")
	if err != nil {
		panic(err)
	}

	c, err := os.ReadFile(clientKey)
	if err != nil {
		panic(err)
	}
	tss2Key, err := keyfile.Decode(c)
	if err != nil {
		panic(err)
	}

	signer, err := tss2Key.Signer(transport.FromReadWriteCloser(tpm), []byte(""), []byte(""))
	if err != nil {
		panic(err)
	}

	clientCert := loadCert(clientCert)
	caCert, err := os.ReadFile(caCert)
	if err != nil {
		panic(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsCert := tls.Certificate{
		Certificate: [][]byte{clientCert.Raw}, // Leaf cert first
		PrivateKey:  signer,                   // The TPM-backed signer!
		Leaf:        clientCert,               // Optional optimization
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		RootCAs:      caCertPool,
		//ServerName:   "localhost",
	}

	httpTransport := &http.Transport{
		TLSClientConfig: tlsCfg,
	}
	client := &http.Client{Transport: httpTransport}
	fmt.Println("creating request")
	resp, err := client.Get("https://" + serverEndpoint + "/" + clientKey)
	//resp, err := client.Post("https://"+serverEndpoint+"/edge-cert", "text/html", bytes.NewBuffer([]byte("Hello, World!")))
	if err != nil {
		log.Fatalf("error client.Get: %v", err)
	}
	fmt.Println(resp.Status)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(respBody))
	resp.Body.Close()
}
func loadCert(cert string) *x509.Certificate {
	// --- 4. Load Client Certificate ---
	certPEM, err := os.ReadFile(cert)
	if err != nil {
		log.Fatalf("Failed to read client certificate file: %v", err)
	}
	certDER, _ := pem.Decode(certPEM)
	if certDER == nil {
		log.Fatalf("Failed to decode PEM block from")
	}
	leafCert, err := x509.ParseCertificate(certDER.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse client certificate: %v", err)
	}
	log.Printf("Loaded client certificate: Subject=%q, Issuer=%q", leafCert.Subject, leafCert.Issuer)
	return leafCert
}

var TPMDEVICES = []string{"/dev/tpm0", "/dev/tpmrm0"}

func OpenTPM(path string) (io.ReadWriteCloser, error) {
	if slices.Contains(TPMDEVICES, path) {
		return tpmutil.OpenTPM(path)
	} else if path == "simulator" {
		return simulator.GetWithFixedSeedInsecure(1073741825)
	} else {
		return net.Dial("tcp", path)
	}
}
