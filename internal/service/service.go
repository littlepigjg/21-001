package service

import (
	"benzhi/internal/config"
	"benzhi/internal/store"
	"benzhi/pkg/tokenizer"
)

type Service struct {
	store *store.Store

	cfg config.Config

	tokenizer *tokenizer.Tokenizer
}

func New(st *store.Store, cfg config.Config) *Service {
	return &Service{
		store:     st,
		cfg:       cfg,
		tokenizer: tokenizer.New(),
	}
}

func (s *Service) Store() *store.Store {
	return s.store
}
