# Day 12: testes e validações

Objetivo do dia: escrever os testes que o CI do contrib exige, com as
ferramentas que o repositório usa. Sem isso seu PR não passa, e mais importante,
sem isso seu componente quebra em produção.

Os exemplos ficam em exemplos/. Cada arquivo de teste demonstra uma técnica.

```bash
cd exemplos
go test ./...
go test -bench=. -benchmem ./...
```

## O que o CI cobra

Antes das técnicas, a lista do que precisa existir:

* Cobertura razoável do código novo. Não existe número mágico, mas caminho de
  erro sem teste é apontado na revisão.
* Teste de ciclo de vida: o componente sobrevive a Start seguido de Shutdown.
* Teste de goleak: nenhuma goroutine sobrevive aos testes.
* Teste de config carregada de arquivo, incluindo os casos que devem falhar.
* Testes rodando com o detector de corrida ligado.

## 1. Table-driven, o formato padrão

Um slice de casos, um subteste por caso:

```go
tests := []struct {
    name     string
    cfg      Config
    entrada  map[string]string
    esperado map[string]string
}{
    {name: "remove chave", ...},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { ... })
}
```

A vantagem prática não é estética: quando um caso quebra, a saída do go test já
diz qual foi, e adicionar um caso novo é acrescentar uma linha.

Sobre testify, que é a biblioteca usada em todo o repositório:

* require aborta o teste na hora. Use quando continuar não faz sentido, como um
  erro inesperado ou um ponteiro nil.
* assert continua. Use para verificar vários campos e ver todos os problemas de
  uma vez.

A regra prática: require para pré-condições, assert para verificações.

Arquivo: exemplos/sanitizer_test.go

## 2. Config carregada de arquivo

Testar a struct em Go direto deixa passar erro de tag mapstructure, que é
justamente o erro que o usuário vai encontrar. Carregue de um YAML:

```go
cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
sub, err := cm.Sub("valido")
var cfg Config
require.NoError(t, sub.Unmarshal(&cfg))
```

O padrão do repositório é um arquivo em testdata com um bloco por cenário,
incluindo os que devem falhar na validação. Teste o Validate junto:

```go
require.ErrorIs(t, cfg.Validate(), ErrSemChaves)
```

Use ErrorIs contra um erro sentinela, não Contains contra a mensagem. Mensagem
muda, identidade não.

Arquivo: exemplos/config_test.go e exemplos/testdata/config.yaml

## 3. Os helpers de teste do core

Existem para você não montar mocks na mão:

componenttest.NewNopHost para o parâmetro host do Start.

componenttest.CheckConfigStruct valida as regras que o repositório exige de
qualquer struct de config, como tag mapstructure em todo campo exportado.

exportertest.NewNopSettings, receivertest.NewNopSettings,
processortest.NewNopSettings, connectortest.NewNopSettings e
extensiontest.NewNopSettings constroem as Settings da classe.

consumertest.NewNop devolve um consumer que descarta tudo.

consumertest.TracesSink, MetricsSink e LogsSink guardam tudo o que recebem. São
a ferramenta principal para testar receiver, processor e connector: você empurra
dados e inspeciona o sink.

```go
sink := new(consumertest.TracesSink)
p, _ := factory.CreateTraces(ctx, processortest.NewNopSettings(typeStr), cfg, sink)
require.NoError(t, p.ConsumeTraces(ctx, entrada))
require.Len(t, sink.AllTraces(), 1)
```

Os exemplos dos dias 07 a 11 usam esses helpers.

## 4. Nada de sleep

Esperar telemetria com time.Sleep fixo deixa o teste lento e instável. Use
Eventually, que tenta de novo até o prazo:

```go
require.Eventually(t, func() bool {
    return sink.LogRecordCount() >= 2
}, 2*time.Second, 10*time.Millisecond)
```

Arquivo: day08/tickreceiver/receiver_test.go

## 5. goleak

Falha o pacote inteiro se alguma goroutine sobreviver aos testes:

```go
func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

É a rede de proteção contra o bug mais comum de componente: subir um loop no
Start e esquecer de encerrá-lo no Shutdown. No contrib esse arquivo é gerado
pelo mdatagen com o nome generated_package_test.go e não se edita à mão.

Arquivo: exemplos/package_test.go

## 6. Golden files

Quando a saída esperada é grande, comparar campo a campo vira um teste ilegível.
A alternativa é versionar a saída num arquivo e comparar tudo de uma vez:

```go
esperado, err := golden.ReadTraces(filepath.Join("testdata", "traces-esperado.yaml"))
require.NoError(t, ptracetest.CompareTraces(esperado, saida,
    ptracetest.IgnoreStartTimestamp(),
    ptracetest.IgnoreEndTimestamp(),
))
```

Os pacotes são pkg/golden e pkg/pdatatest, ambos do contrib. As opções Ignore
existem para campos que mudam a cada execução: timestamp, id gerado
aleatoriamente, ordem dos elementos.

Para regerar os arquivos, a convenção é uma flag:

```bash
go test -run TestGoldenTraces -update-golden ./...
```

Duas advertências. A chamada de escrita falha o teste de propósito, para você
não deixar o modo de atualização ligado no commit. E sempre revise o diff do
golden antes de commitar: golden regerado sem revisão transforma um bug em
comportamento esperado.

Arquivo: exemplos/golden_test.go

## 7. Benchmarks

Componente roda uma vez por span, então caminho quente merece medição:

```bash
go test -bench=. -benchmem -run=XXX ./...
```

```
BenchmarkProcessTraces-24   66874   16297 ns/op   33184 B/op   407 allocs/op
```

A coluna de allocs por operação costuma ser a mais reveladora. Ao comparar duas
versões, use benchstat em vez de olhar os números a olho nu.

Cuidado clássico: se o benchmark muta o dado, clone dentro do laço. Sem isso a
primeira iteração faz o trabalho e as seguintes medem o nada.

## 8. Corrida de dados

O CI roda com o detector ligado, e você também deve rodar:

```bash
go test -race ./...
```

Ele só detecta corrida em código que efetivamente executou concorrente, então
vale escrever um teste que empurra dados de várias goroutines quando o
componente tem estado compartilhado.

## Antes de abrir o PR

```bash
make lint
make test
make generate
git diff --exit-code   # nada de arquivo gerado fora do commit
```

## Checklist do dia

* Escrevo testes table-driven com require e assert nos lugares certos.
* Carrego config de testdata, incluindo os casos inválidos.
* Uso sinks e Eventually em vez de mocks feitos à mão e sleep.
* Tenho goleak no pacote e rodo os testes com race.

Próximo: [Day 13, boas práticas dos mantenedores](../day13/README.md)
