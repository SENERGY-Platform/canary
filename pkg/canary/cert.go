/*
 * Copyright 2025 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package canary

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/SENERGY-Platform/cert-certificate-authority/pkg/client"
)

var ErrNewCertNeeded = errors.New("new cert needed")

func (this *Canary) getTlsConfig(token string, hubId string, exp time.Duration) (*tls.Config, error) {
	cert, err := this.getCert(token, hubId, exp)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}

func (this *Canary) getCert(token string, hubId string, exp time.Duration) (cert tls.Certificate, err error) {
	cert, err = this.loadClientCertFromFile()
	if errors.Is(err, ErrNewCertNeeded) {
		err = this.loadNewCertsToFiles(token, hubId, exp)
		if err != nil {
			return cert, err
		}
		return this.loadClientCertFromFile()
	}
	return this.loadClientCertFromFile()
}

func (this *Canary) loadClientCertFromFile() (cert tls.Certificate, err error) {
	keyPath := this.config.CertKeyFilePath
	certPath := this.config.CertFilePath
	if _, err = os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
		return cert, errors.Join(ErrNewCertNeeded, err)
	}
	if _, err = os.Stat(certPath); errors.Is(err, os.ErrNotExist) {
		return cert, errors.Join(ErrNewCertNeeded, err)
	}
	cert, err = tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Println("ERROR: tls.LoadX509KeyPair()", err)
		return cert, errors.Join(ErrNewCertNeeded, err)
	}
	if cert.Leaf != nil && !cert.Leaf.NotAfter.IsZero() && cert.Leaf.NotAfter.Before(time.Now()) {
		return cert, fmt.Errorf("%w: cert expired", ErrNewCertNeeded)
	}
	return cert, nil
}

func (this *Canary) loadNewCertsToFiles(token string, hubId string, exp time.Duration) error {
	key, cert, _, err := client.NewClient(this.config.CertAuthorityUrl).NewCertAndKey(pkix.Name{}, []string{hubId}, exp, &token)
	if err != nil {
		return err
	}
	keyPem, err := privateKeyToPemBlock(key)
	if err != nil {
		return err
	}
	certPem := certToPemBlock(cert)
	return writeKeyAndCertPemFiles(this.config.CertKeyFilePath, this.config.CertFilePath, keyPem, certPem)
}

func privateKeyToPemBlock(key any) (*pem.Block, error) {
	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: b,
	}, nil
}

func certToPemBlock(cert *x509.Certificate) *pem.Block {
	return &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
}

func writePemFile(pth string, block *pem.Block, perm os.FileMode) error {
	file, err := os.OpenFile(pth, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer file.Close()
	return pem.Encode(file, block)
}

func writeKeyAndCertPemFiles(keyPath string, certPath string, keyBlock *pem.Block, certBlock *pem.Block) error {
	err := writePemFile(keyPath, keyBlock, 0600)
	if err != nil {
		return err
	}
	err = writePemFile(certPath, certBlock, 0600)
	if err != nil {
		return err
	}
	return nil
}
