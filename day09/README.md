# Day 09: como criar um processor

Objetivo do dia: escrever um processor e entender a única armadilha que
realmente machuca nessa classe, que é a mutação de dados compartilhados.

O processor deste dia se chama tag. Ele adiciona atributos no Resource de tudo
que passa. Código em tagprocessor/.

```bash
cd tagprocessor
go test ./...
```

## O que um processor é

O único componente que fica no meio: consome de quem está atrás e produz para
quem está na frente. A assinatura no helper deixa isso explícito:

```go
type ProcessTracesFunc func(context.Context, ptrace.Traces) (ptrace.Traces, error)
```

Recebe os dados, devolve os dados.

## MutatesData, a parte que importa

Se o seu processor escreve qualquer coisa dentro do pdata que recebeu, ele
precisa declarar:

```go
processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true})
```

Por quê: pdata é passado por referência. Quando um receiver alimenta dois
pipelines, os dois recebem o mesmo objeto. Se um pipeline tem um componente que
muta e o Collector não sabe disso, o outro pipeline vê a alteração. O resultado
é um bug que só aparece em produção, com configuração específica, e que é
horrível de reproduzir.

Declarando true, o Collector clona os dados antes de distribuir. Custa memória,
por isso não se declara à toa, mas declarar de menos é infinitamente pior que
declarar demais.

Teste prático: se o seu processor chama qualquer método que começa com Set,
Put, Remove ou Append no pdata recebido, é true.

## Descartar dados

Devolver um erro comum propaga o erro pipeline acima. Para descartar o lote de
propósito, sem virar erro, existe um sentinela:

```go
return td, processorhelper.ErrSkipProcessingData
```

É o que um processor de amostragem ou de filtro usa.

## Ordem no pipeline

A ordem dos processors é a ordem da lista dentro do pipeline. Isso não é
detalhe de configuração, é parte do projeto do seu componente. Duas
consequências:

Se o seu processor depende de um atributo que outro adiciona, documente a ordem
esperada no README. Muita gente vai errar.

memory_limiter costuma vir primeiro e batch por último. Se o seu processor é
caro, colocá-lo depois do batch reduz o número de invocações, mas atrasa a
decisão.

## Custo por dado

O processor roda uma vez por lote, e normalmente uma vez por span, log ou data
point dentro dele. É o lugar do pipeline onde um desperdício pequeno vira custo
grande. Três hábitos:

Pré-compute na construção o que não muda. No exemplo, a lista de chaves da
config vira um mapa uma vez só, e não a cada span.

Evite alocar dentro do laço. Reaproveite buffers e evite fmt.Sprintf em caminho
quente.

Escreva um benchmark. O day12 mostra como.

## OTTL antes de escrever código

Antes de criar um processor novo, verifique se o transformprocessor e o
filterprocessor já resolvem seu caso. Eles usam OTTL, uma linguagem de
transformação, e cobrem uma faixa enorme de necessidades sem componente novo:

```yaml
processors:
  transform:
    trace_statements:
      - set(resource.attributes["env"], "prod")
```

Um mantenedor vai fazer exatamente essa pergunta no seu PR de doação. Ter a
resposta pronta economiza semanas.

## Vendo funcionar

No builder-config.yaml do day05:

```yaml
processors:
  - gomod: github.com/apolzek/otel-collector-contrib-journey/day09/tagprocessor v0.0.0

replaces:
  - github.com/apolzek/otel-collector-contrib-journey/day09/tagprocessor => ../../day09/tagprocessor
```

No config.yaml:

```yaml
processors:
  tag:
    attributes:
      cluster: prod-1
    overwrite: false

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [tag]
      exporters: [print]
```

## Exercícios

1. Faça o processor descartar spans com duração menor que um limite, usando
   ErrSkipProcessingData, e teste o caminho.
2. Adicione suporte a metrics. Repare que Resource fica em ResourceMetrics, a
   estrutura se repete.
3. Escreva um benchmark e compare a versão com mapa pré-computado contra uma
   que percorre o slice da config a cada span.

## Checklist do dia

* Sei decidir se meu processor precisa de MutatesData true, e sei o porquê.
* Sei descartar dados sem gerar erro no pipeline.
* Penso no custo por span antes de escrever o laço.
* Sei responder por que o transformprocessor não resolveria meu caso.

Próximo: [Day 10, como criar uma extension](../day10/README.md)
