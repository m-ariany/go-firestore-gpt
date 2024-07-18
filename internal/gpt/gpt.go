package gpt

import (
	"sync"

	gpt "github.com/m-ariany/gpt-chat-client"
)

var (
	client *gpt.Client
	once   sync.Once
)

type ClientFactory interface {
	Client() (Client, error)
	ClientWithConfig(ClientConfig) (Client, error)
}

type factory struct {
}

func NewClientFactory(cnf ClientConfig) (ClientFactory, error) {
	var err error
	once.Do(func() {
		client, err = gpt.NewClient(cnf)
	})
	return &factory{}, err
}

func (g factory) Client() (Client, error) {
	return Client{Client: client.Clone()}, nil
}

func (g factory) ClientWithConfig(cnf ClientConfig) (Client, error) {
	return Client{Client: client.CloneWithConfig(cnf)}, nil
}

type Client struct {
	*gpt.Client
}

type ClientConfig = gpt.ClientConfig
