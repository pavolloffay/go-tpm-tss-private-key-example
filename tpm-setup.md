# Generate Cerficates using TPM
```bash
sudo tpm2_createprimary -c primary.ctx
sudo tpm2_create --parent-context primary.ctx --public sub.pub --private sub.priv

sudo tpm2_load --parent-context primary.ctx --public sub.pub --private sub.priv --key-context sub.ctx                                                                                                                                                                                                                                                 ploffay@fedora
name: 000ba93b00fffb9f3121bc24062dc2230b9e49c32afe395ae47150f501983f2f86cb

sudo openssl dgst -sha1 -binary -out hash.bin msg.txt
```

https://github.com/aws/rolesanywhere-credential-helper/pull/38/files

## Installation of TPM

To install and verify TPM usage, make sure, `secure boot` and `Trusted Platform Module` is enabled while booting VM. 

`Ubuntu 20.04` OS is used to build and verify TPM usage. Also, TPM stack supports `openssl <= v1.1.x`. To support `openssl3`, its required to use `tpm2-openssl` stack, this has not been used/verified as part of this sample.

*Note: `Ubuntu 20.04` comes with `openssl v1.1.1`, by default. If you are using Ubuntu >20.4 or anyother OS, please make sure that you are `openssl v1.1.x`*

### Install TPM stack
```bash
sudo apt-get update

sudo apt-get install -y \
  autoconf-archive \
  libcmocka0 \
  libcmocka-dev \
  procps \
  iproute2 \
  build-essential \
  git \
  pkg-config \
  gcc \
  libtool \
  automake \
  libssl-dev \
  uthash-dev \
  autoconf \
  doxygen \
  libjson-c-dev \
  libini-config-dev \
  libcurl4-openssl-dev \
  uuid-dev \
  libltdl-dev \
  libusb-1.0-0-dev
```

```bash
git clone https://github.com/tpm2-software/tpm2-tss.git

cd tpm2-tss
./bootstrap
./configure
sudo make clean; make -j 4
sudo make install
```
### Install TPM engine
```bash
git clone --branch 1.2.0 https://github.com/tpm2-software/tpm2-tss-engine.git

cd tpm2-tss-engine
./bootstrap
./configure
sudo make clean; make -j 4
sudo make install
```

### Install TPM  tools

```bash
git clone --branch 5.5 https://github.com/tpm2-software/tpm2-tools.git

cd tpm2-tools
./bootstrap
./configure
sudo make clean; make -j 4
sudo make install
```

```bash
ldconfig
```

## Generate Asymetric key pair using TPM

```bash
sudo tpm2tss-genkey -a rsa -s 2048 edge-cert
```

```bash
openssl rsa -engine tpm2tss -inform engine -in edge-cert -pubout -outform pem -out edge-cert.pub

# OR
sudo openssl rsa -provider tpm2 -in edge-cert -pubout -outform pem -out edge-cert.pub
```


## Generate certificate using self-signed CA

* Create self-signed CA certificate

```bash
openssl genrsa -des3 -out ca.key 2048

```

```bash
openssl req -new -x509 -days 1826 -key ca.key -out ca.crt -subj "/C=US/ST=CA/L=Santa Clara/O=Edge/OU=Edge/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
openssl x509 -in ca.crt -out ca.pem -outform PEM
```

* Creat csr using openssl. Here openssl uses `tpm2tss` engine.
```bash
sudo openssl req -new -engine tpm2tss -keyform engine -key edge-cert -out edge-cert.csr -subj "/C=US/ST=CA/L=Santa Clara/O=Edge/OU=Edge/CN=localhost"

# This worked
sudo openssl req -new -provider tpm2  -key edge-cert -out edge-cert.csr -subj "/C=US/ST=CA/L=Santa Clara/O=Edge/OU=Edge/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

* Sign `edge-cert` using self-signed CA
```bash
openssl x509 -req -in edge-cert.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out edge-cert.crt -days 1826 -copy_extensions copyall
```
## Verification of TPM certificate usage (mTLS)

### Launch local sample web server using `openssl`
```bash
 openssl genrsa -out server.key 2048

 openssl req -new -key server.key -out server.csr  -subj "/C=US/ST=CA/L=Santa Clara/O=Edge/OU=Edge/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

 openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 360 -copy_extensions copyall

 openssl s_server -accept 9443 -cert server.crt -key server.key -CAfile ca.pem -Verify 1
```

### Launch openssl client (without edge certificate) - expected to fail mTLS authentication as client doesn't present its certificate
```bash
openssl s_client  -CAfile ca.crt -connect localhost:9443
```
### Launch openssl client with TPM engine and using edge certificate
```bash
openssl s_client  -keyform engine  -engine tpm2tss -cert edge-cert.crt -key edge-cert -CAfile ca.crt -connect localhost:9443

## This works
sudo openssl s_client   -provider tpm2 -cert edge-cert.crt -key edge-cert -CAfile ca.crt -connect localhost:9443
```