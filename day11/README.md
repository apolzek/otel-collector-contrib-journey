# Day 11: como criar um connector

Objetivo do dia: escrever um connector e entender por que ele é a classe mais
fácil de configurar errado.

O connector deste dia se chama spancount. Ele lê um pipeline de traces e produz
uma métrica com a contagem de spans. Código em spancountconnector/.

```bash
cd spancountconnector
go test ./...
```

## O que um connector é

Um componente que é exporter de um pipeline e receiver de outro, ao mesmo tempo.
Ele liga dois pipelines dentro do mesmo Collector, sem passar pela rede.

Serve para três coisas na prática:

Derivar um sinal de outro. Métricas a partir de spans é o caso clássico, e é o
que o spanmetricsconnector faz.

Rotear. Mandar telemetria para pipelines diferentes conforme um atributo, que é
o routingconnector.

Encadear processamento. Passar por uma sequência de processors, ramificar e
processar de novo de formas diferentes.

## Declarado por par de sinais

Esta é a particularidade central da classe. Um connector não suporta sinais,
suporta PARES de sinais:

```go
connector.WithTracesToMetrics(createTracesToMetrics, component.StabilityLevelDevelopment)
```

As combinações possíveis são todas: traces para traces, traces para metrics,
traces para logs, metrics para qualquer coisa, logs para qualquer coisa. Cada
par suportado é uma opção na factory, uma função de criação e uma entrada de
estabilidade no metadata.yaml.

O tipo devolvido diz de qual lado ele consome:

```go
func createTracesToMetrics(
    _ context.Context,
    _ connector.Settings,
    cfg component.Config,
    next consumer.Metrics,
) (connector.Traces, error)
```

Devolve connector.Traces, que é Component mais consumer.Traces, porque é assim
que ele aparece no pipeline de traces. Recebe consumer.Metrics, que é a saída
para o pipeline de destino.

## O erro de configuração que todo mundo comete

Um connector aparece DUAS vezes no YAML, e as duas com o mesmo nome:

```yaml
connectors:
  spancount:
    metric_name: spans.count

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [spancount]      # aqui ele é exporter
    metrics:
      receivers: [spancount]      # aqui ele é receiver
      exporters: [print]
```

Se ele aparecer só como exporter, os dados entram e não saem para lugar nenhum.
Se aparecer só como receiver, o pipeline de destino nunca recebe nada. Nos dois
casos o Collector sobe sem reclamar e você fica olhando para um dashboard vazio.

E não pode haver ciclo. Um connector que consome do pipeline A e produz no
mesmo pipeline A é detectado na montagem do grafo e derruba o boot.

## Cuidados de implementação

Não emita telemetria vazia. Um lote que não gerou nada deve retornar sem chamar
o próximo consumer. Ponto de dados zero enviado a cada lote polui o backend e
custa dinheiro.

Escolha a temporalidade conscientemente. O exemplo usa Delta, que reporta o
que aconteceu neste lote. Cumulative exige manter estado entre chamadas, e aí
você precisa pensar em memória, em reinício do processo e em expiração de séries
que pararam de aparecer.

Cardinalidade é o risco número um. Copiar um atributo do span para a métrica
parece inofensivo até alguém colocar um id de usuário ali. Documente quais
dimensões são seguras e imponha um limite.

Estado compartilhado precisa de proteção. O ConsumeTraces pode ser chamado de
várias goroutines ao mesmo tempo.

## Vendo funcionar

No builder-config.yaml do day05:

```yaml
connectors:
  - gomod: github.com/apolzek/otel-collector-contrib-journey/day11/spancountconnector v0.0.0

replaces:
  - github.com/apolzek/otel-collector-contrib-journey/day11/spancountconnector => ../../day11/spancountconnector
```

Use o YAML da seção anterior no config.yaml e mande um span com o curl do
day01. A contagem sai pelo pipeline de metrics.

## Exercícios

1. Acrescente uma dimensão de status, contando separadamente spans com erro.
2. Troque Delta por Cumulative e resolva os problemas que aparecem: onde guardar
   o estado, o que acontece no restart, quando expirar uma série.
3. Adicione o par traces para logs, emitindo um log por span com erro.

## Checklist do dia

* Sei que connector é declarado por par de sinais.
* Sei escrever o YAML com o connector nos dois pipelines.
* Não emito telemetria vazia nem exploto a cardinalidade.
* Entendo a diferença prática entre Delta e Cumulative.

Próximo: [Day 12, testes e validações](../day12/README.md)
