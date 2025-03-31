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

## TPM commands for wrapped key

In the context of a TPM, a "wrapped key" refers to a cryptographic key (its sensitive private portion) that has been encrypted using another key, typically a "parent" key that resides within or is managed by the TPM. This allows the wrapped key to be stored securely outside the TPM (e.g., on disk) because it can only be decrypted (unwrapped) and used by the TPM itself, under the correct conditions and with the correct parent key.

Here are the key TPM 2.0 commands involved in managing wrapped keys:

1. TPM2_Create / TPM2_CreateLoaded:
Purpose: To create a new key object (like an RSA or ECC key pair, or a symmetric key) under a specified parent key.
Relevance to Wrapping: When TPM2_Create successfully generates a new key pair, it returns the public portion of the key directly. Crucially, it returns the private portion encrypted ("wrapped") by the public part of the specified parent key. This wrapped private blob is what you would typically store externally.
TPM2_CreateLoaded does the same but also loads the newly created key into the TPM immediately, returning a handle for use, in addition to the wrapped private/public parts.

2. TPM2_Load:
Purpose: To load a key (that was previously created using TPM2_Create) back into the TPM using its parent.
Relevance to Wrapping: This command takes the public portion and the wrapped private portion of a key, along with the handle of its parent key. The TPM uses the parent key (internally) to decrypt ("unwrap") the private portion and load the key into one of its volatile slots. If successful, it returns a handle that can be used in subsequent cryptographic operations (like signing or decryption). This is the primary way to make an externally stored wrapped key usable again.

3. TPM2_Import:
Purpose: To import a key into the TPM that was either generated externally or duplicated using TPM2_Duplicate.
Relevance to Wrapping: This command is used when the key's private portion is wrapped differently than the standard output of TPM2_Create. Specifically, when using TPM2_Duplicate, the private key is encrypted with a symmetric key, which itself is wrapped by the new parent's public key. TPM2_Import takes these components (public key, duplicated private blob, encryption key, optional symmetric algorithm definition) and the new parent handle to unwrap and load the key.

4. TPM2_Duplicate:
Purpose: To "re-wrap" an existing key (that is already loaded in the TPM) under a different parent key. This is often used for migrating keys between TPMs or between different hierarchies within the same TPM.
Relevance to Wrapping: It takes the handle of the key to be duplicated and the handle of the new parent. It encrypts (wraps) the private portion of the target key, typically using a symmetric key provided (or generated internally), and then encrypts that symmetric key using the public portion of the new parent. It outputs these wrapped components, ready to be imported using TPM2_Import under the new parent.

5. TPM2_LoadExternal:
Purpose: To load a public key, or potentially a key where the private portion is not TPM-wrapped (e.g., a key from a file without TPM protection), into the TPM.
Relevance to Wrapping: This is generally not used for standard wrapped keys protected by TPM parents. It's more for bringing external public keys into the TPM for verification tasks or potentially for keys managed outside the TPM's wrapping mechanisms.

## TPM how to sign with wrapped key

1. Load the Wrapped Key: Provide the wrapped private key blob, its corresponding public key, and identify its parent key to the TPM. The TPM uses the parent key (which must already be loaded or available) to decrypt (unwrap) the private key and load it into a temporary, secure slot. The TPM returns a handle (a reference) to this now-active key.
2. Perform the Signing Operation: Instruct the TPM to use the handle of the loaded key to sign the data (or typically, the hash of the data) you provide, using a specified signing scheme (e.g., RSASSA-PKCS1-v1_5, ECDSA). The TPM performs the calculation internally using the unwrapped private key.
3. Receive the Signature: The TPM returns the resulting digital signature. The private key itself never leaves the TPM boundary during this process.
4. (Optional but Recommended) Flush the Key: Release the key handle and clear the key from the TPM's active memory slot to free up resources.


Let's assume you have:

* parent.ctx: The context file for the parent key (already loaded or made persistent).
* mykey.pub: The public part of the key you want to use for signing.
* mykey.priv: The wrapped private part of the key (output from tpm2_create).
* data_to_sign.txt: The file containing the data you want to sign.
Here's how you would typically proceed:

1. Load the Wrapped Key:

Use tpm2_load to load the public (mykey.pub) and wrapped private (mykey.priv) parts under the parent key (parent.ctx).
This command unwraps the private key inside the TPM and saves the context (including the handle) of the now-loaded key to an output file (e.g., mykey.ctx).
Bash

```bash
# Command syntax: tpm2_load -C <parent_context> -u <public_key_file> -r <private_key_file> -c <output_key_context_file>
tpm2_load -C parent.ctx -u mykey.pub -r mykey.priv -c mykey.ctx
Note: If the key has an authorization password, you'll need to provide it using -p or other mechanisms during loading and signing.
```

2. Sign the Data:
Use tpm2_sign with the loaded key context (mykey.ctx).
Provide the data file (data_to_sign.txt). The tool usually hashes the data internally first before passing the hash to the TPM for signing (check the tool's documentation for specifics or use -d for pre-hashed digests).
Specify the output file for the signature (signature.bin).
You might need to specify the signing scheme (e.g., -g sha256 -s rsassa).
Bash

```bash
# Command syntax: tpm2_sign -c <key_context_file> -g <hash_algorithm> -s <signing_scheme> -o <output_signature_file> <input_data_file>
# Example for RSA key:
tpm2_sign -c mykey.ctx -g sha256 -s rsassa -o signature.bin data_to_sign.txt


# Example for ECC key (scheme often determined automatically or use appropriate flag):
# tpm2_sign -c mykey.ctx -g sha256 -o signature.bin data_to_sign.txt
```

3. Flush the Key Context (Optional but Recommended):
Use tpm2_flushcontext to remove the key from the TPM's active memory.
Bash

```bash
# Command syntax: tpm2_flushcontext <context_file_to_flush OR handle>
tpm2_flushcontext mykey.ctx
# Alternatively, if you know the handle (e.g., 0x81000001): tpm2_flushcontext 0x81000001
After flushing, the mykey.ctx file is no longer valid. You would need to run tpm2_load again to use the key.
```

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
