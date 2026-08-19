# Day 02: core e contrib, dois repositórios

Objetivo do dia: saber onde cada coisa mora, por que o projeto é dividido em
dois repositórios e como navegar no monorepo do contrib sem se perder.

## Os repositórios

opentelemetry-collector, o core:
https://github.com/open-telemetry/opentelemetry-collector

opentelemetry-collector-contrib, os componentes da comunidade:
https://github.com/open-telemetry/opentelemetry-collector-contrib

opentelemetry-collector-releases, quem monta e publica os binários:
https://github.com/open-telemetry/opentelemetry-collector-releases

## O que o core é

O core é o motor, não a coleção de componentes. Ele traz as regras do jogo que
todos os 300 e poucos componentes do contrib precisam seguir:

* component, o contrato mínimo de qualquer componente.
* pdata, o modelo de dados em memória para traces, metrics, logs e profiles.
* consumer, as interfaces que ligam um componente no próximo.
* confmap, a camada de configuração: providers, resolução de referências,
  merge, unmarshal e validação.
* service, o orquestrador que monta o grafo e controla o ciclo de vida.
* os helpers que evitam reimplementar o difícil: exporterhelper (fila,
  retry, timeout), processorhelper, receiverhelper, scraper.
* featuregate, para ligar e desligar mudanças de comportamento em deploy.
* cmd/builder, o OCB, que gera distribuições.

De componentes prontos, o core traz só o mínimo para ser útil sozinho:

| classe | componentes |
|---|---|
| receiver | otlp, nop |
| processor | batch, memory_limiter, queuebatch |
| exporter | otlp, otlphttp, debug, nop |
| extension | zpages, memory_limiter |
| connector | forward |

Uma correção que evita confusão: o repositório core não publica binário. Ele
tem o pacote otelcol, que é a biblioteca para montar um Collector, e
cmd/otelcorecol, um build de desenvolvimento usado para testar o próprio core.
A distribuição que você baixa como .deb, .rpm ou imagem Docker vem do
repositório releases.

## O que o contrib é

Tudo que fala com o mundo real. Na versão v0.159.0, as pastas de primeiro
nível têm aproximadamente:

| classe | pastas de primeiro nível |
|---|---|
| receiver | 114 |
| exporter | 47 |
| processor | 35 |
| extension | 32 |
| connector | 14 |

Algumas dessas pastas agrupam mais componentes dentro, como extension/storage
e extension/observer, então o total real é maior. Além disso existem pkg
(bibliotecas compartilhadas, entre elas o OTTL), internal, cmd e testbed.

O ponto estrutural que muda tudo: cada componente é um módulo Go independente,
com go.mod próprio, versionado e publicado separadamente. O contrib tem mais de
trezentos arquivos go.mod. Isso significa que:

* Você mexe num componente sem recompilar o repositório inteiro.
* Cada componente escolhe suas dependências. O kafkaexporter puxa o franz-go,
  e quem não usa Kafka não carrega essa dependência.
* Cada módulo tem um Makefile de uma linha que inclui o Makefile.Common da
  raiz, herdando os alvos padrão (make lint, make test, make generate).

## Como um componente é organizado

Abra qualquer pasta de componente do contrib e você encontra sempre a mesma
estrutura:

| arquivo | função |
|---|---|
| factory.go | NewFactory, a config padrão e as funções de criação por sinal |
| config.go | a struct Config, as tags mapstructure e o Validate |
| <nome>.go | a implementação em si |
| metadata.yaml | a fonte da verdade: tipo, estabilidade por sinal, codeowners, telemetria interna |
| doc.go | doc do pacote e a diretiva go:generate que chama o mdatagen |
| README.md | documentação para o usuário final |
| documentation.md | gerado a partir do metadata.yaml, não edite |
| internal/metadata/ | gerado pelo mdatagen |
| generated_*_test.go | gerados: ciclo de vida e goleak |
| testdata/ | YAMLs de configuração usados nos testes |
| go.mod, go.sum, Makefile | o módulo em si |

Regra prática: qualquer arquivo com generated_ no nome, mais o
documentation.md, sai de make generate a partir do metadata.yaml. Se precisar
mudar, mexa no YAML e regenere.

## Versionamento: por que o go.mod mistura v1 e v0

O core segue o VERSIONING.md do projeto: as APIs consideradas estáveis vão para
módulos v1.x (component, pdata, consumer, confmap) e o que ainda pode mudar
fica em v0.x (exporterhelper, processorhelper, otelcol, connector).

Por isso um go.mod de componente tem linhas assim:

```
go.opentelemetry.io/collector/component v1.65.0
go.opentelemetry.io/collector/exporter/exporterhelper v0.159.0
```

As duas são a mesma release. A regra de bolso: v1.65.0 e v0.159.0 andam juntas,
e você nunca deve misturar releases diferentes no mesmo build.

O projeto libera uma versão a cada duas semanas mais ou menos, e core, contrib
e releases sobem juntos com o mesmo número de v0.

## Como achar as coisas no contrib

Você vai passar mais tempo lendo esse repositório do que escrevendo nele.
Algumas rotas que economizam tempo:

* Para entender uma classe de componente, leia um componente pequeno e estável
  antes de um grande. O debugexporter e o forwardconnector cabem na cabeça.
* Para descobrir quem é dono de um componente, abra o metadata.yaml dele. Os
  codeowners estão ali, e são as pessoas que vão revisar seu PR.
* Para saber se um componente é confiável em produção, olhe stability e
  distributions no metadata.yaml, não o README.
* Para achar exemplo de uma API do core, procure o uso dela no contrib com
  grep. Trezentos componentes é uma base de exemplos enorme.

## Checklist do dia

* Sei dizer o que está no core e o que está no contrib.
* Entendo por que cada componente é um módulo Go separado.
* Sei ler um metadata.yaml e tirar dele estabilidade e codeowners.
* Não me assusto mais com v1.65.0 e v0.159.0 no mesmo go.mod.

Próximo: [Day 03, Go essencial parte 1](../day03/README.md)
