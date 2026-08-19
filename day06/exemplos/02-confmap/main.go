// confmap é a camada de configuração do Collector. Ela resolve de onde o texto
// vem (arquivo, env, http), expande as referências ${...}, faz merge de vários
// arquivos e só então despeja o resultado na struct de Config.
//
// Rode com: go run ./02-confmap
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
)

type TLSConfig struct {
	Insecure bool `mapstructure:"insecure"`
}

type Config struct {
	Endpoint string            `mapstructure:"endpoint"`
	Timeout  time.Duration     `mapstructure:"timeout"`
	Headers  map[string]string `mapstructure:"headers"`
	TLS      TLSConfig         `mapstructure:"tls"`

	_ struct{}
}

var errEndpointVazio = errors.New("endpoint é obrigatório")

// Se a struct implementa confmap.Validator, o Collector chama Validate antes
// de criar o componente. Erro aqui significa Collector que não sobe.
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return errEndpointVazio
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout precisa ser positivo, veio %s", c.Timeout)
	}
	return nil
}

func main() {
	os.Setenv("TENANT", "vindo-do-ambiente")

	resolver, err := confmap.NewResolver(confmap.ResolverSettings{
		URIs: []string{"file:02-confmap/config.yaml"},
		ProviderFactories: []confmap.ProviderFactory{
			fileprovider.NewFactory(),
			envprovider.NewFactory(),
		},
		DefaultScheme: "env",
	})
	if err != nil {
		panic(err)
	}

	conf, err := resolver.Resolve(context.Background())
	if err != nil {
		panic(err)
	}

	// Os defaults vêm primeiro. O YAML é despejado POR CIMA, então tudo que o
	// usuário não escreveu mantém o padrão da factory.
	cfg := Config{
		Timeout: 5 * time.Second,
		Headers: map[string]string{},
	}
	if err := conf.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", cfg)
	fmt.Println("validação:", cfg.Validate())

	// Campo desconhecido é ERRO, não aviso. Um typo no YAML derruba o boot em
	// vez de virar silêncio.
	fmt.Println("--- typo no YAML ---")
	comTypo := confmap.NewFromStringMap(map[string]any{
		"endpoint": "http://x",
		"timeoutt": "10s",
	})
	var alvo Config
	fmt.Println("erro:", comTypo.Unmarshal(&alvo))

	// Merge: é assim que se separa base e overlay por ambiente sem template.
	fmt.Println("--- merge ---")
	base := confmap.NewFromStringMap(map[string]any{
		"endpoint": "http://base:4318",
		"timeout":  "5s",
	})
	overlay := confmap.NewFromStringMap(map[string]any{
		"endpoint": "http://prod:4318",
	})
	if err := base.Merge(overlay); err != nil {
		panic(err)
	}
	var merged Config
	if err := base.Unmarshal(&merged); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", merged)
}
