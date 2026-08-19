// O padrão Factory é o coração do Collector. Um registro global mapeia um nome
// vindo do YAML para uma função que sabe construir aquele componente.
//
// Rode com: go run ./01-factory
package main

import (
	"fmt"
	"sort"
)

// Config é qualquer coisa. No Collector, component.Config é literalmente
// "type Config any".
type Config any

// Factory é o contrato: quem sou eu, qual é minha config padrão e como me
// construir.
type Factory interface {
	Type() string
	CreateDefaultConfig() Config
	Create(cfg Config) (Component, error)
}

type Component interface {
	Descrever() string
}

// ---------------------------------------------------------------------------
// A implementação genérica da Factory, escrita UMA vez. Nenhum componente
// implementa a interface Factory na mão: todos chamam este construtor.
// ---------------------------------------------------------------------------

type criarDefaultFunc func() Config
type criarFunc func(Config) (Component, error)

type factory struct {
	tipo         string
	criarDefault criarDefaultFunc
	criar        criarFunc
}

func (f *factory) Type() string                { return f.tipo }
func (f *factory) CreateDefaultConfig() Config { return f.criarDefault() }
func (f *factory) Create(c Config) (Component, error) {
	return f.criar(c)
}

func NewFactory(tipo string, d criarDefaultFunc, c criarFunc) Factory {
	return &factory{tipo: tipo, criarDefault: d, criar: c}
}

// ---------------------------------------------------------------------------
// Um componente concreto. Isto é tudo o que um autor de componente escreve.
// ---------------------------------------------------------------------------

type ConfigArquivo struct {
	Caminho string
}

type exportadorArquivo struct{ cfg ConfigArquivo }

func (e *exportadorArquivo) Descrever() string {
	return "escrevo em " + e.cfg.Caminho
}

func NewFactoryArquivo() Factory {
	return NewFactory("arquivo",
		func() Config { return ConfigArquivo{Caminho: "/tmp/otel.log"} },
		func(c Config) (Component, error) {
			cfg, ok := c.(ConfigArquivo)
			if !ok {
				return nil, fmt.Errorf("config do tipo errado: %T", c)
			}
			return &exportadorArquivo{cfg: cfg}, nil
		},
	)
}

// ---------------------------------------------------------------------------
// O registro. É o components.go que o OCB gera para a sua distribuição.
// ---------------------------------------------------------------------------

func registrar(fs ...Factory) map[string]Factory {
	m := make(map[string]Factory, len(fs))
	for _, f := range fs {
		m[f.Type()] = f
	}
	return m
}

func main() {
	registro := registrar(NewFactoryArquivo())

	nomes := make([]string, 0, len(registro))
	for n := range registro {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)
	fmt.Println("componentes disponíveis:", nomes)

	// Isto é, em miniatura, o que o Collector faz ao ler o YAML: acha a
	// factory pelo nome, pega os defaults, aplica o que o usuário escreveu
	// por cima e só então constrói.
	f := registro["arquivo"]
	cfg := f.CreateDefaultConfig().(ConfigArquivo)
	cfg.Caminho = "/var/log/telemetria.log" // o que veio do YAML

	comp, err := f.Create(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println(comp.Descrever())

	// Tipo desconhecido: exatamente o erro "unknown type" que o Collector
	// devolve quando o componente não está no binário.
	if _, ok := registro["kafka"]; !ok {
		fmt.Println(`erro: tipo desconhecido "kafka"`)
	}
}
