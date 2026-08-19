# otel-collector-contrib-journey

Um manual de 19 dias para sair do zero e conseguir contribuir com o
OpenTelemetry Collector e com o repositório
[opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib).

Cada dia é uma pasta com um README e exemplos que rodam de verdade. Todo o
código foi compilado e testado contra o core v1.65.0 e v0.159.0.

## Os dias

| dia | assunto |
|---|---|
| [01](day01/README.md) | O que é o Collector: pipeline, sinais e a primeira configuração rodando |
| [02](day02/README.md) | Core e contrib: os dois repositórios, o monorepo e o versionamento |
| [03](day03/README.md) | Go essencial 1: interfaces, embedding, asserção de tipo, erros, struct tags |
| [04](day04/README.md) | Go essencial 2: factory, functional options, concorrência, context, generics |
| [05](day05/README.md) | OCB: montando sua própria distribuição |
| [06](day06/README.md) | As interfaces e os helpers do Collector: component, consumer, pdata, confmap |
| [07](day07/README.md) | Como criar um exporter |
| [08](day08/README.md) | Como criar um receiver |
| [09](day09/README.md) | Como criar um processor |
| [10](day10/README.md) | Como criar uma extension |
| [11](day11/README.md) | Como criar um connector |
| [12](day12/README.md) | Testes e validações |
| [13](day13/README.md) | As boas práticas que os mantenedores cobram |
| [14](day14/README.md) | Do zero ao hero: doando um componente |
| [15](day15/README.md) | OTTL, a linguagem de transformação do Collector |
| [16](day16/README.md) | mdatagen: o metadata.yaml que vira código, teste e doc |
| [17](day17/README.md) | Resiliência e performance: fila, retry, timeout, profiling |
| [18](day18/README.md) | O dia a dia no monorepo e o seu primeiro PR |
| [19](day19/README.md) | Operação, troubleshooting e segurança |

## Como usar

Os dias 01 a 14 são o caminho principal, em ordem: os dias 03 e 04 sustentam
tudo que vem depois, e os dias 07 a 11 assumem o vocabulário do dia 06.

Os dias 15 a 19 são aprofundamento e podem ser lidos fora de ordem, conforme a
necessidade:

* 15 quando for mexer em transformação de dados.
* 16 na primeira vez que criar ou editar um metadata.yaml.
* 17 quando o componente precisar aguentar backend instável.
* 18 antes do primeiro PR.
* 19 quando precisar operar ou depurar um Collector.

Se você já sabe Go bem, pule 03 e 04. Se já conhece o Collector como usuário,
comece no 02.

## Pré-requisitos

* Go 1.25 ou mais novo
* Docker, usado nos dias 01 e 19
* curl

## Rodando os exemplos

Cada pasta de exemplo é um módulo Go independente, igual aos componentes do
contrib. Os exemplos de teoria rodam direto:

```bash
cd day03/exemplos && go run ./01-interfaces
cd day04/exemplos && go run ./01-factory
cd day06/exemplos && go run ./01-pdata
```

Os componentes têm suíte de testes:

```bash
cd day07/printexporter && go test ./...
cd day08/tickreceiver  && go test ./...
cd day09/tagprocessor  && go test ./...
cd day10/heartbeatextension && go test ./...
cd day11/spancountconnector && go test ./...
cd day12/exemplos && go test ./... && go test -bench=. -benchmem ./...
cd day15/exemplos && go run ./01-executando && go test ./...
cd day16/queuewatchreceiver && go test ./...
cd day17/resilienteexporter && go test -v ./...
cd day18/exemplos && go run .
```

E o dia 05 compila um Collector de verdade com o exporter do dia 07 dentro:

```bash
go install go.opentelemetry.io/collector/cmd/builder@v0.159.0
cd day05
builder --config=builder-config.yaml
./_build/meu-otelcol --config=config.yaml
```

## O que você vai construir

| dia | componente | o que faz |
|---|---|---|
| 07 | printexporter | escreve uma linha por span e por log |
| 08 | tickreceiver | gera um log a cada intervalo |
| 09 | tagprocessor | adiciona atributos no Resource |
| 10 | heartbeatextension | toca um arquivo periodicamente e expõe um contador |
| 11 | spancountconnector | conta spans e emite uma métrica |
| 15 | função OTTL mascarar | mascara o valor de um atributo |
| 16 | queuewatchreceiver | receiver de scrape com código gerado por mdatagen |
| 17 | resilienteexporter | exporter com fila, retry e timeout de verdade |

Todos são pequenos de propósito. O objetivo é que cada um caiba na cabeça e
mostre a particularidade da sua classe, não que sirvam em produção.

## Licença

[MIT](LICENSE)
