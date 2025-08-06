package config

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt"
	"io/ioutil"
)

func GetPrivateKey(path string) (*rsa.PrivateKey, error) {
	privateKeyFile, err := ioutil.ReadFile(path)
	if err != nil {
		msg := fmt.Sprintf("Could not read private file from path %s", path)
		return nil, errors.New(msg)
	}
	rsaPri, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyFile)
	if err != nil {
		msg := fmt.Sprintf("Could not uderstand the Private Key file, %s", err.Error())
		return nil, errors.New(msg)
	}
	return rsaPri, nil
}

func GetPublicKey(path string) (*rsa.PublicKey, error) {
	publicKeyFile, err := ioutil.ReadFile(path)
	if err != nil {
		msg := fmt.Sprintf("Could not read public file from path %s", path)
		return nil, errors.New(msg)
	}
	rsaPub, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyFile)
	if err != nil {
		msg := fmt.Sprintf("Could not uderstand the Public Key file, %s", err.Error())
		return nil, errors.New(msg)
	}
	return rsaPub, nil
}