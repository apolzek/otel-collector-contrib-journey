# Day 19: operação, troubleshooting e segurança

Objetivo do dia: enxergar o Collector como quem opera. Isso muda como você
escreve componente, porque você passa a saber o que a pessoa do outro lado
precisa ver quando algo quebra às três da manhã.

Dois arquivos aqui: troubleshooting.yaml para rodar agora na sua máquina, e
gateway.yaml como referência de produção.

## Agent e gateway

Duas topologias, e quase toda instalação usa as duas:

Agent é um Collector por host ou por pod, colado na aplicação. Ele coleta
localmente, enriquece com metadados de infraestrutura e manda para o gateway.
Fica pequeno de propósito.

Gateway é um pool central que recebe dos agents. É onde ficam as decisões caras
e centralizadas: amostragem, roteamento, redação de dado sensível, fila
persistente e a conexão autenticada com o backend.

Consequência para quem escreve componente: se o seu componente é caro ou
precisa de estado global, ele pertence ao gateway. Se ele depende de estar no
mesmo host que a aplicação, pertence ao agent. Diga isso no README.

## Investigando na prática

Suba o Collector com a configuração de troubleshooting:

```bash
docker run --rm -d --name otelcol-day19 \
  -p 14318:4318 -p 13133:13133 -p 55679:55679 -p 1777:1777 -p 8888:8888 \
  -v "$PWD/troubleshooting.yaml:/etc/otelcol-contrib/config.yaml" \
  otel/opentelemetry-collector-contrib:latest
```

Mande um span:

```bash
curl -X POST http://127.0.0.1:14318/v1/traces \
  -H 'Content-Type: application/json' -d @../day01/span-exemplo.json
```

Agora as quatro janelas de diagnóstico.

Está vivo:

```bash
curl -s http://127.0.0.1:13133/
```

```
{"status":"Server available","upSince":"...","uptime":"2.25s"}
```

Os pipelines montaram como você esperava:

```bash
curl -s http://127.0.0.1:55679/debug/pipelinez
```

O zpages também tem /debug/tracez, com spans recentes do próprio Collector, e
/debug/servicez.

O dado está entrando e saindo:

```bash
curl -s http://127.0.0.1:8888/metrics | grep -E "^otelcol_"
```

```
otelcol_receiver_accepted_spans{receiver="otlp",transport="http"} 1
otelcol_exporter_sent_spans{exporter="debug"} 1
otelcol_process_uptime 2.269
```

Onde está gastando memória:

```bash
go tool pprof -http=:8080 http://127.0.0.1:1777/debug/pprof/heap
```

Limpe depois:

```bash
docker rm -f otelcol-day19
```

## As métricas que respondem as perguntas de verdade

Quando alguém diz que a telemetria sumiu, estas quatro localizam o problema em
menos de um minuto:

| métrica | pergunta que responde |
|---|---|
| otelcol_receiver_accepted_spans e refused | o dado chegou? foi recusado na entrada? |
| otelcol_processor_dropped_spans | algum processor derrubou? normalmente o memory_limiter |
| otelcol_exporter_sent_spans e send_failed_spans | saiu? o backend recusou? |
| otelcol_exporter_queue_size contra queue_capacity | a fila está enchendo? |

O padrão de leitura: accepted alto e sent baixo significa problema para a
frente, ou seja, backend ou fila. accepted zero significa problema para trás,
ou seja, cliente, rede ou porta.

Isto tem efeito direto no seu componente: se ele não reporta essa telemetria,
ninguém consegue diagnosticá-lo. É por isso que o day16 insiste no mdatagen e
no bloco telemetry.

## O debug exporter

A ferramenta mais rápida para responder o que exatamente está passando:

```yaml
exporters:
  debug:
    verbosity: detailed
```

São três níveis: basic mostra contagem, normal mostra um resumo, detailed
despeja a telemetria inteira. Nunca deixe detailed ligado em produção, porque o
volume de log fica maior que o volume de telemetria.

## Segurança

Segredos. Use configopaque.String nos campos de token, senha e chave. O tipo
imprime asteriscos em log e em dump de configuração. Um exporter que declara
o token como string comum vaza credencial no primeiro erro logado.

Origem do segredo. No YAML, use `${env:BACKEND_TOKEN}` ou o provider de arquivo,
nunca o valor literal. O gateway.yaml deste diretório faz assim.

TLS. Use configtls no seu componente, que já traz ca_file, cert_file, key_file,
insecure e insecure_skip_verify com os nomes que todo mundo conhece. Nunca
deixe insecure como padrão.

Autenticação. Não implemente autenticação dentro do seu exporter. Use uma
extension de auth, via confighttp ou configgrpc. Assim o seu componente aceita
qualquer mecanismo, inclusive os que ainda não existem.

Dado sensível. A telemetria carrega mais PII do que as pessoas imaginam: header
de autorização, query string com token, corpo de log com email. Redija no
gateway, antes de sair do seu perímetro:

```yaml
processors:
  transform:
    trace_statements:
      - delete_key(span.attributes, "http.request.header.authorization")
      - set(span.attributes["user.email"], "[REDACTED]") where span.attributes["user.email"] != nil
```

Existe também o redactionprocessor no contrib, com listas de bloqueio e de
permissão.

Superfície exposta. Endpoint com 0.0.0.0 escuta em todas as interfaces. pprof e
zpages nunca devem estar expostos publicamente. No gateway.yaml, o pprof está
em 127.0.0.1 de propósito.

Distribuição enxuta. Um binário feito com OCB só com o que você usa tem menos
CVE para responder do que o contrib inteiro. É o argumento de segurança do
day05.

## Antes de subir para produção

```bash
otelcol validate --config=config.yaml
```

Valide em CI, não em produção. Config inválida derruba o boot, e derrubar o
boot no deploy é bem melhor do que descobrir depois.

O gateway.yaml deste diretório passa nessa validação e junta o que este dia
discutiu: memory_limiter primeiro, redação com transform, batch por último,
fila persistente com storage, extensions de diagnóstico e telemetria interna
exposta em Prometheus.

## Exercícios

1. Rode o troubleshooting.yaml, mande um span e encontre todas as quatro
   métricas da tabela acima.
2. Force o pipeline a falhar apontando o exporter para um endereço morto e veja
   send_failed subir e a fila encher.
3. Tire um perfil de heap e ache a maior alocação.

## Checklist do dia

* Sei diagnosticar um Collector com zpages, métricas internas e pprof.
* Sei quais quatro métricas respondem onde a telemetria sumiu.
* Sei proteger segredos com configopaque e variáveis de ambiente.
* Penso em como o meu componente vai ser diagnosticado por outra pessoa.

Voltar ao [índice](../README.md)
