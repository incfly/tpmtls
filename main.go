// Binary tpmtls is a POC for TPM-based TLS.
package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"

	"github.com/awly/tpmtls/tpmkey"
	"github.com/google/go-tpm/tpm2"
)

var (
	iter = flag.Int("iter", 1000, "the number of the iterations")
)

func main() {
	flag.Parse()
	pk, err := tpmkey.PrimaryECC("/dev/tpm0", tpm2.HandleOwner)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer pk.Close()

	fmt.Println("loadKey OK")

	crt, err := createClientCert(pk)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("createClientCert OK")

	srv, err := createServer()
	if err != nil {
		fmt.Println(err)
		return
	}
	go runServer(srv)
	fmt.Println("runServer OK")
	defer srv.Close()

	for i := 0; i < *iter; i++ {
		fmt.Printf("making connection %v\n", i)
		conn, err := makeConnection(srv, crt)
		if err != nil {
			fmt.Println("tls.Dial:", err)
			return
		}
		if _, err := conn.Write([]byte("hi")); err != nil {
			fmt.Println("Write:", err)
			return
		}
		resp := make([]byte, 1024)
		_, err = conn.Read(resp)
		if err != nil {
			fmt.Println("Read:", err)
			return
		}
		conn.Close()
	}
}

func makeConnection(srv net.Listener, clientCert *tls.Certificate) (*tls.Conn, error) {
	conn, err := tls.Dial("tcp", srv.Addr().String(), &tls.Config{
		InsecureSkipVerify: true,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return clientCert, nil
		},
	})
	if err != nil {
		fmt.Println("tls.Dial:", err)
		return nil, err
	}
	return conn, nil
}

func createClientCert(pk crypto.Signer) (*tls.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %s", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, pk.Public(), pk)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %s", err)
	}

	return &tls.Certificate{PrivateKey: pk, Leaf: template, Certificate: [][]byte{derBytes}}, nil
}

func createServerCert() (*tls.Certificate, error) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %s", err)
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %s", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, pk.Public(), pk)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %s", err)
	}

	return &tls.Certificate{PrivateKey: pk, Leaf: template, Certificate: [][]byte{derBytes}}, nil
}

func createServer() (net.Listener, error) {
	crt, err := createServerCert()
	if err != nil {
		return nil, err
	}

	lis, err := tls.Listen("tcp", ":0", &tls.Config{
		Certificates: []tls.Certificate{*crt},
		ClientAuth:   tls.RequireAnyClientCert,
	})
	if err != nil {
		return nil, err
	}
	return lis, nil
}

func runServer(l net.Listener) {
	for {
		clientConn, err := l.Accept()
		if err != nil {
			fmt.Println("Accept Error:", err)
			return
		}
		go func(conn net.Conn) {
			// TODO: figure out why connection is the contraint here, we run too fast?
			fmt.Println("Handle Connection")
			io.Copy(conn, conn)
			conn.Close()
		}(clientConn)
	}
}
