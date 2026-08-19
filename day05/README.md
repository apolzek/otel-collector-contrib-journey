# Day 05: OCB, o OpenTelemetry Collector Builder

Objetivo do dia: montar sua própria distribuição do Collector, contendo só os
componentes que você usa, incluindo um componente local ainda não publicado.

## Por que existe

O binário otelcol-contrib traz trezentos e poucos componentes. Em produção isso
significa imagem grande, tempo de deploy maior e superfície de ataque bem além
do necessário para quem só usa três componentes.

O OCB resolve isso invertendo a lógica: em vez de tirar o que não presta, você
lista o que quer. Ele gera o código Go de um Collector com exatamente esses
módulos e compila.

O que ele gera é um components.go assim:

```go
factories.Receivers, err = otelcol.MakeFactoryMap[receiver.Factory](
    otlpreceiver.NewFactory(),
)
```

Um mapa de tipo para factory. É por isso que a factory é a peça central da
arquitetura: é a única coisa que um componente precisa expor para ser plugável.
Todo o resto do dia 4 existe para sustentar essa linha.

## Instalação

```bash
go install go.opentelemetry.io/collector/cmd/builder@v0.159.0
```

O binário se chama builder e reporta a versão como ocb:

```bash
builder version
```

Use sempre a mesma versão do builder e dos módulos que você vai listar. Versões
diferentes geram conflito de dependência na hora de compilar.

## A receita

O arquivo builder-config.yaml deste diretório é o exemplo completo. As partes:

```yaml
dist:
  name: meu-otelcol
  output_path: ./_build
  otelcol_version: 0.159.0
```

O bloco dist define o nome do binário, onde o projeto gerado é escrito e qual
release do core usar. Depois vem uma lista por classe de componente:

```yaml
receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.159.0

exporters:
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.159.0
  - gomod: github.com/apolzek/otel-collector-contrib-journey/day07/printexporter v0.0.0

replaces:
  - github.com/apolzek/otel-collector-contrib-journey/day07/printexporter => ../../day07/printexporter
```

Cada entrada gomod é o caminho do módulo seguido da versão, exatamente como
apareceria num go.mod.

O bloco replaces é o que torna o OCB útil enquanto você desenvolve. Ele aponta
um módulo para um caminho no disco, então dá para testar seu componente num
Collector real antes de publicar qualquer coisa. Sem isso o Go tentaria baixar
o módulo do proxy e falharia.

## Rodando

```bash
builder --config=builder-config.yaml
./_build/meu-otelcol --config=config.yaml
```

Em outro terminal, mande um span:

```bash
curl -X POST http://127.0.0.1:4318/v1/traces \
  -H 'Content-Type: application/json' \
  -d @../day01/span-exemplo.json
```

A saída esperada vem do printexporter que você vai escrever no day07:

```
[meu-otelcol] span service=checkout name=POST /checkout trace_id=5b8ef... duration=1.5s
```

O diretório _build é gerado e está no .gitignore. Ele contém main.go,
components.go, go.mod e o binário. Vale abrir o components.go pelo menos uma
vez: é a peça que conecta tudo que você estudou até aqui.

## Opções que valem conhecer

* --skip-compilation gera o código sem compilar. Útil para inspecionar o
  components.go ou para compilar depois com flags próprias.
* --skip-get-modules pula o go get, se você já tem tudo em cache.
* dist.debug_compilation gera binário com símbolos de debug, para usar delve.

## Erros comuns

O Collector sobe mas diz unknown type. O componente não está no build. Confira
se ele aparece na lista certa do builder-config.yaml e se você recompilou.

Erro de versão incompatível na compilação. Alguma entrada gomod está numa
release diferente do otelcol_version. Alinhe todas.

O replace não funcionou. O caminho é relativo ao output_path, não ao
builder-config.yaml. No exemplo, _build fica em day05/_build, então o caminho
para o day07 sobe dois níveis.

## Onde o OCB aparece de novo

Os dias 07 a 11 escrevem um componente cada. Em todos eles o jeito de ver o
componente funcionando de verdade é este: acrescentar uma entrada gomod mais um
replace neste builder-config.yaml e rodar de novo.

## Checklist do dia

* Montei uma distribuição própria e rodei ela.
* Sei plugar um componente local por replace, sem publicar nada.
* Sei explicar o que tem dentro do components.go gerado.
* Sei por que se usa OCB em produção em vez do contrib inteiro.

Próximo: [Day 06, interfaces e helpers do Collector](../day06/README.md)
