# Day 01: o que é o OpenTelemetry Collector

Objetivo do dia: entender o problema que o Collector resolve, o modelo de
pipeline e conseguir subir um Collector recebendo telemetria de verdade.

## O problema

Sem Collector, cada aplicação instrumentada manda telemetria direto para o
backend de observabilidade. Isso cria três dores:

* Cada aplicação precisa saber o endereço, o formato e as credenciais do
  backend. Trocar de backend significa redeploy de tudo.
* Cada aplicação faz retry, buffer e compressão por conta própria, mal e em
  cinco linguagens diferentes.
* Não existe um ponto único para filtrar dado sensível, reduzir volume ou
  enriquecer com metadados de infraestrutura.

O Collector é um processo que fica no meio. As aplicações mandam para ele, ele
manda para o backend. É um proxy de telemetria com pipeline programável.

## O modelo de pipeline

Cinco classes de componente. Decore estas cinco linhas, o resto do repositório
gira em torno delas:

| classe | papel | posição |
|---|---|---|
| receiver | entrada, escuta uma porta ou faz scrape | produz |
| processor | transforma, filtra, agrupa, limita | meio |
| exporter | saída, manda para um backend | consome |
| connector | liga a saída de um pipeline na entrada de outro | exporter de um lado, receiver do outro |
| extension | capacidade que não toca em telemetria | fora do pipeline |

Duas dimensões se cruzam nessas classes.

A primeira é o sinal: traces, metrics, logs e profiles (este último ainda em
desenvolvimento). Um pipeline é sempre de um sinal só, não existe pipeline
misto. Um mesmo componente pode suportar vários sinais, com nível de
estabilidade diferente em cada um.

A segunda é a instância. Em otlp e otlp/interno você tem duas instâncias do
mesmo tipo com configurações diferentes. O nome completo, no formato
tipo/nome, é o que o Collector chama de component.ID.

## Anatomia de uma configuração

```yaml
receivers:
  otlp:
    protocols: { http: { endpoint: 0.0.0.0:4318 } }

processors:
  batch:

exporters:
  debug:
    verbosity: detailed

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
```

Os blocos de cima declaram instâncias. O bloco service liga as instâncias em
pipelines. São coisas separadas, e é aí que mora a maior parte dos enganos de
quem começa:

* Declarar um componente no bloco de cima não liga ele em nada. Se ele não
  aparecer em service.pipelines, simplesmente não é instanciado.
* A ordem dos processors é a ordem da lista dentro do pipeline, não a ordem em
  que aparecem no bloco processors. A recomendação padrão é memory_limiter
  primeiro (derrubar dados antes de estourar a memória) e batch por último.
* Um receiver pode alimentar vários pipelines, e vários receivers podem
  alimentar um só. Fan-in e fan-out são nativos, sem nada extra.
* Campo desconhecido no YAML é erro, não aviso. Um typo em verbosiy derruba o
  boot em vez de virar silêncio.

## Mão na massa

Neste diretório estão config.yaml e span-exemplo.json.

Suba um Collector com a imagem oficial do contrib:

```bash
docker run --rm -d --name otelcol-day01 -p 4318:4318 \
  -v "$PWD/config.yaml:/etc/otelcol-contrib/config.yaml" \
  otel/opentelemetry-collector-contrib:latest
```

Mande um span pela API OTLP/HTTP:

```bash
curl -X POST http://127.0.0.1:4318/v1/traces \
  -H 'Content-Type: application/json' \
  -d @span-exemplo.json
```

Veja o span chegar do outro lado:

```bash
docker logs otelcol-day01 | grep -A 6 "POST /checkout"
docker rm -f otelcol-day01
```

Antes de subir, dá para checar a configuração sem executar nada:

```bash
docker run --rm -v "$PWD/config.yaml:/tmp/config.yaml" \
  otel/opentelemetry-collector-contrib:latest validate --config=/tmp/config.yaml
```

Esse comando validate é o mesmo que roda no boot: ele resolve a config,
faz o unmarshal em cada struct de componente e chama o Validate de cada uma.

## Experimente quebrar

Aprender o que dá erro vale tanto quanto aprender o que funciona:

1. Troque verbosity por verbosiy e rode o validate. Leia a mensagem.
2. Tire debug da lista exporters do pipeline, mas deixe declarado em cima.
   O Collector sobe? O que ele reclama?
3. Adicione um exporter chamado kafka sem ter o componente no binário.
   O erro é unknown type, e ele vem da resolução do grafo.

## Checklist do dia

* Sei explicar em duas frases por que o Collector existe.
* Sei a diferença entre declarar um componente e ligar ele num pipeline.
* Sei que pipeline é sempre de um sinal só.
* Consigo subir um Collector e ver telemetria entrar e sair.

Próximo: [Day 02, core e contrib](../day02/README.md)
