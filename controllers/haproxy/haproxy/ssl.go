package haproxy

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/glog"

	"github.com/reAsOn2010/ingress/controllers/haproxy/haproxy/config"
)

// SSLCert describes a SSL certificate to be used in Haproxy
type SSLCert struct {
	CertFileName string
	KeyFileName  string
	// PemFileName contains the path to the file with the certificate and key concatenated
	PemFileName string
	// PemSHA contains the sha1 of the pem file.
	// This is used to detect changes in the secret that contains the certificates
	PemSHA string
	// CN contains all the common names defined in the SSL certificate
	CN []string
}

// AddOrUpdateCertAndKey creates a .pem file wth the cert and the key with the specified name
func (ha *Manager) AddOrUpdateCertAndKey(name string, cert string, key string) (SSLCert, error) {
	pemFileName := config.SSLDirectory + "/" + name + ".pem"

	pem, err := os.Create(pemFileName)
	if err != nil {
		return SSLCert{}, fmt.Errorf("Couldn't create pem file %v: %v", pemFileName, err)
	}
	defer pem.Close()

	_, err = pem.WriteString(fmt.Sprintf("%v\n%v", cert, key))
	if err != nil {
		return SSLCert{}, fmt.Errorf("Couldn't write to pem file %v: %v", pemFileName, err)
	}

	cn, err := ha.commonNames(pemFileName)
	if err != nil {
		return SSLCert{}, err
	}

	return SSLCert{
		CertFileName: cert,
		KeyFileName:  key,
		PemFileName:  pemFileName,
		PemSHA:       ha.pemSHA1(pemFileName),
		CN:           cn,
	}, nil
}

// commonNames checks if the certificate and key file are valid
// returning the result of the validation and the list of hostnames
// contained in the common name/s
func (ha *Manager) commonNames(pemFileName string) ([]string, error) {
	pemCerts, err := ioutil.ReadFile(pemFileName)
	if err != nil {
		return []string{}, err
	}

	block, _ := pem.Decode(pemCerts)
	if block == nil {
		return []string{}, fmt.Errorf("No valid PEM formatted block found")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return []string{}, err
	}

	cn := []string{cert.Subject.CommonName}
	if len(cert.DNSNames) > 0 {
		cn = append(cn, cert.DNSNames...)
	}

	glog.V(2).Infof("DNS %v %v\n", cn, len(cn))
	return cn, nil
}

// SearchDHParamFile iterates all the secrets mounted inside the base directory
// in order to find a file with the name dhparam.pem. If such file exists it will
// returns the path. If not it just returns an empty string
func (ha *Manager) SearchDHParamFile(baseDir string) string {
	dhPath := fmt.Sprintf("%v/dhparam.pem", baseDir)
	if _, err := os.Stat(dhPath); err == nil {
		glog.Infof("using file '%v' for parameter ssl_dhparam", dhPath)
		return dhPath
	}

	glog.Warning("no file dhparam.pem found in secrets")
	return ""
}

func (ha *Manager) pemSHA1(filename string) string {
	hasher := sha1.New()
	s, err := ioutil.ReadFile(filename)
	if err != nil {
		return ""
	}

	hasher.Write(s)
	return hex.EncodeToString(hasher.Sum(nil))
}
