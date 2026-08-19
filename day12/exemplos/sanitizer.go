// Package exemplos tem um componente propositalmente pequeno. O que interessa
// aqui são os TESTES ao lado dele, um para cada técnica que o repositório do
// Collector usa.
package exemplos

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type Config struct {
	// RemoveKeys são atributos apagados de todo span.
	RemoveKeys []string `mapstructure:"remove_keys"`

	// Placeholder substitui o valor em vez de apagar a chave.
	Placeholder string `mapstructure:"placeholder"`

	_ struct{}
}

var ErrSemChaves = errors.New("remove_keys não pode ser vazio")

func (c *Config) Validate() error {
	if len(c.RemoveKeys) == 0 {
		return ErrSemChaves
	}
	return nil
}

type Sanitizer struct {
	chaves map[string]struct{}
	cfg    *Config
}

func NewSanitizer(cfg *Config) *Sanitizer {
	chaves := make(map[string]struct{}, len(cfg.RemoveKeys))
	for _, k := range cfg.RemoveKeys {
		chaves[k] = struct{}{}
	}
	return &Sanitizer{chaves: chaves, cfg: cfg}
}

func (s *Sanitizer) ProcessTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	for _, rs := range td.ResourceSpans().All() {
		for _, ss := range rs.ScopeSpans().All() {
			for _, span := range ss.Spans().All() {
				s.limpar(span)
			}
		}
	}
	return td, nil
}

func (s *Sanitizer) limpar(span ptrace.Span) {
	attrs := span.Attributes()
	if s.cfg.Placeholder != "" {
		for k := range s.chaves {
			if _, ok := attrs.Get(k); ok {
				attrs.PutStr(k, s.cfg.Placeholder)
			}
		}
		return
	}
	attrs.RemoveIf(func(k string, _ pcommon.Value) bool {
		_, remover := s.chaves[k]
		return remover
	})
}
