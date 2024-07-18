package config

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/caarlos0/env/v8"
)

type GilasAI struct {
	ApiKey string `env:"GILAS_API_KEY,required"`
	ApiUrl string `env:"GILAS_API_URL" envDefault:"https://api.gilas.io/v1/chat/completions"`
	Model  string `env:"GILAS_GPT_MODEL" envDefault:"gpt-3.5-turbo"`
}

type Firebase struct {
	Type                    string        `env:"FIREBASE_TYPE,required" json:"type"`
	ProjectId               string        `env:"FIREBASE_PROJECT_ID,required" json:"project_id"`
	PrivateKeyId            string        `env:"FIREBASE_PRIVATE_KEY_ID,required" json:"private_key_id"`
	PrivateKey              string        `env:"FIREBASE_PRIVATE_KEY,required" json:"private_key"`
	ClientEmail             string        `env:"FIREBASE_CLIENT_EMAIL,required" json:"client_email"`
	ClientId                string        `env:"FIREBASE_CLIENT_ID,required" json:"client_id"`
	AuthUri                 string        `env:"FIREBASE_AUTH_URI,required" json:"auth_uri"`
	TokenUri                string        `env:"FIREBASE_TOKEN_URI,required" json:"token_uri"`
	AuthProviderX509CertUrl string        `env:"FIREBASE_AUTH_PROVIDER_X509_CERT_URL,required" json:"auth_provider_x509_cert_url"`
	ClientX509CertUrl       string        `env:"FIREBASE_CLIENT_X509_CERT_URL,required" json:"client_x509_cert_url"`
	WriteTimeoutSecond      time.Duration `env:"FIREBASE_WRITE_TIMEOUT_SECOND"`
}

type Youtube struct {
	ApiKey string `env:"YOUTUBE_API_KEY"`
}

type Config struct {
	GilasAI
	Firebase
	Youtube
}

func LoadConfigOrPanic() Config {
	var config *Config = new(Config)
	if err := env.Parse(config); err != nil {
		panic(err)
	}

	config.normalize()
	return *config
}

func (c *Config) normalize() {

	decodedBytes, err := base64.StdEncoding.DecodeString(c.Firebase.PrivateKey)
	if err != nil {
		panic(err)
	}
	c.Firebase.PrivateKey = string(decodedBytes)
	c.Firebase.PrivateKey = strings.ReplaceAll(c.Firebase.PrivateKey, "\\n", "\n")

	if c.WriteTimeoutSecond == 0 {
		c.WriteTimeoutSecond = time.Second * 30
	}
}
