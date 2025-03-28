# Golang trusted platform module (TPM) TSS2 private key example

## TSS2 private key explanation

* The SRK is a primary, asymmetric key pair (public and private) managed by the Trusted Platform Module (TPM).
* Its main purpose is to protect other keys created by applications or the operating system (often called "child keys").
* When a new key needs to be stored securely but cannot remain loaded inside the limited TPM memory, it can be "wrapped" (encrypted) using the SRK's public key.
* This wrapped key can then be safely stored outside the TPM (e.g., on disk).
* Only the same TPM that wrapped the key, using its internal SRK private key, can "unwrap" (decrypt) it later to load it back into the TPM for use.
* The SRK private key, like other primary TPM private keys, is designed to never leave the TPM chip.

## Go implementation

* Get crypto.Signer: Obtain an object that implements the crypto.Signer interface using the loaded TPM key. go-tpm-tools provides helpers for this.
* Configure tls.Config:
* Create a tls.Certificate struct.
* Populate its Certificate field with the raw bytes of your client certificate chain (leaf first).
* Assign the TPM-backed crypto.Signer object to the PrivateKey field.
* Set RootCAs to specify the CAs trusted for verifying the server's certificate (or use InsecureSkipVerify only for testing).

## Generate certificates

The certificates are generated with openssl 3.x and the tpm2 provider - https://github.com/tpm2-software/tpm2-openssl.

The openssl 1.x use tpm2-tss-engine engine https://github.com/tpm2-software/tpm2-tss-engine

```bash
sudo setenforce 0
sudo openssl list  -provider tpm2  -provider default  --providers                                                                                                                                                                                                                                                                                         ploffay@fedora
Providers:
  default
    name: OpenSSL Default Provider
    version: 3.2.4
    status: active
  legacy
    name: OpenSSL Legacy Provider
    version: 3.2.4
    status: active
  tpm2
    name: TPM 2.0 Provider
    version: 1.2.0
    status: active
```

### Generate self-signed CA

```bash
openssl genrsa -des3 -out ca.key 2048
openssl req -new -x509 -days 1826 -key ca.key -out ca.crt -subj "/C=US/ST=CA/L=Santa Clara/O=Edge/OU=Edge/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
openssl x509 -in ca.crt -out ca.pem -outform PEM
```

### Generate TPM private key

```bash
sudo tpm2tss-genkey -a rsa -s 2048 edge-cert.key
# Public cert
# sudo openssl rsa -provider tpm2 -in edge-cert -pubout -outform pem -out edge-cert.pub
# Creat csr using openssl with tpm2 engine
sudo openssl req -new -provider tpm2  -key edge-cert.key -out edge-cert.csr -subj "/C=US/ST=CA/L=Santa Clara/O=Edge/OU=Edge/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
# Sign `edge-cert` with CA
openssl x509 -req -in edge-cert.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out edge-cert.crt -days 1826 -copy_extensions copyall
```

## Run mTLS to verify the connection

### Generate server certs

```bash
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr  -subj "/C=US/ST=CA/L=Santa Clara/O=Edge/OU=Edge/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 360 -copy_extensions copyall
```

### Openssl s_server

```bash
openssl s_server -accept 9443 -cert server.crt -key server.key -CAfile ca.pem -Verify 1 -WWW
```

### Openssl s_client

```bash
sudo openssl s_client  -provider tpm2 -cert edge-cert.crt -key edge-cert.key -CAfile ca.crt -connect localhost:9443
```

### Golang client

```bash
go build main.go
sudo ./main
creating request
200 ok
-----BEGIN TSS2 PRIVATE KEY-----
MIICEgYGZ4EFCgEDoAMBAf8CBEAAAAEEggEYARYAAQALAAYEcgAAABAAEAgAAAEA
AQEAxst5sMgT+d8eVzuPuCPQ3gvlWmwLL/dRSfF/QZUZuI7JiAyu3ljl/zuvXn0P
gg8KzKJY+lw/Sj3Kp+LEw8vpnn3tITxqq2uW/i/vI+c6MlBa7/QBHkxJn/1SoGLP
l61n4CRYwQWvEqI81QCt9KaBs7AOEPusRHlMggrEoag/zMmUJJrLyAukS1r5Xi+F
6g0+3BQhEewa5yt8XO4cMlDvfHk0bE3VhpH++ds8u+WZ8t7DVXyywo2jVyiv/3Fp
y5qZBTLeYkPzz44CTmW2Gf8i3iXd9xPSANhO0I5nhFQtY+dN+agODP/TtVCvi1K3
vGbtYFeWIA1+BUyqA25BIvwR+QSB4ADeACCFzUf59AgrtcMFSKiAj9eAlKdgrqXV
ltuVXzjyuugABwAQqCGLRYolBHnQrqDT+5GF215QSlJC8Z+KizRdrtwn+h9lQCuO
x81fUq1fwEBEFfY+1CPn/fH2IaoBtHgmQkSthuv/1Y2qojkvlELhNlJgrRIeW8zY
TXmZw7jeFJkLPQCA8cnLi6N/mhfYdZEYxMgYq2D4yIjBwif6Yr3xnHcyIo0TXVgF
yQtAh/0W9YZ4NpYrX68jRH9B6ZxX71qz/awooxbrKeNdpO08SGfXdCSTCLmAenk8
v3zO0/EX
-----END TSS2 PRIVATE KEY-----
```

## References

* https://github.com/tpm2-software/tpm2-tools/blob/master/man/tpm2.1.md
* https://github.com/google/go-tpm/tree/main/tpm2
* https://github.com/google/go-tpm-tools
* https://github.com/Foxboron/go-tpm-keyfiles
* https://github.com/aws/rolesanywhere-credential-helper/pull/38
