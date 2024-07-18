package utils

import (
	"github.com/pkoukk/tiktoken-go"
	"github.com/rs/zerolog/log"
)

var tokenizer *tiktoken.Tiktoken

func initTokenizer() error {
	if tokenizer != nil {
		return nil
	}

	tkm, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		log.Error().Err(err).Msg("failed to init tokenizer")
		return err
	}

	tokenizer = tkm

	return nil
}

type Tokenizer struct {
	tokenizer *tiktoken.Tiktoken
}

func NewTokenzier() (Tokenizer, error) {
	if err := initTokenizer(); err != nil {
		return Tokenizer{}, err
	}

	return Tokenizer{tokenizer: tokenizer}, nil
}

func (t Tokenizer) CountTokens(s string) int {
	token := t.tokenizer.Encode(s, nil, nil)
	return len(token)
}
