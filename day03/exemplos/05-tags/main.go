// Struct tags são metadados lidos em runtime por reflexão. A configuração do
// Collector inteira depende disso: a tag mapstructure liga o campo Go ao nome
// no YAML.
//
// Rode com: go run ./05-tags
package main

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// No Collector a tag seria mapstructure. Aqui uso json porque é stdlib e o
// mecanismo é idêntico: uma string lida por reflexão.
type Config struct {
	Endpoint string `json:"endpoint" mapstructure:"endpoint"`
	Timeout  int    `json:"timeout"  mapstructure:"timeout"`

	// Campo sem tag é um erro no Collector: componenttest.CheckConfigStruct
	// reprova a struct.
	Retries int `json:"retries" mapstructure:"retries"`

	// squash embute os campos do struct interno no nível de cima do YAML,
	// em vez de criar um bloco aninhado.
	TLS TLSConfig `json:"tls" mapstructure:",squash"`

	// Campo não exportado é invisível para o unmarshal.
	interno string

	_ struct{}
}

type TLSConfig struct {
	Insecure bool `json:"insecure" mapstructure:"insecure"`
}

func main() {
	entrada := `{"endpoint":"http://localhost:4318","timeout":30,"retries":3,"tls":{"insecure":true}}`

	var cfg Config
	if err := json.Unmarshal([]byte(entrada), &cfg); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", cfg)

	// Isto é, em essência, o que o confmap faz: ler a tag de cada campo para
	// saber qual chave do mapa corresponde a ele.
	fmt.Println("--- tags via reflexão ---")
	t := reflect.TypeOf(cfg)
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Name == "_" {
			continue
		}
		fmt.Printf("campo %-8s tag mapstructure=%q exportado=%v\n",
			f.Name, f.Tag.Get("mapstructure"), f.IsExported())
	}
	_ = cfg.interno
}
