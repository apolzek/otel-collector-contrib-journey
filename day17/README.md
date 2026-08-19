# Day 17: resiliência e performance

Objetivo do dia: o que separa um exporter de exemplo de um exporter que aguenta
o backend cair. E como medir antes de otimizar.

O exemplo deste dia é o resilienteexporter, com fila, retry e timeout de
verdade, e testes que provam cada comportamento.

```bash
cd resilienteexporter
go test -v ./...
```

## As quatro camadas do exporterhelper

Quando você passa as opções de fila, retry e timeout, o helper monta isto, de
fora para dentro:

```
ConsumeTraces
  -> fila (sending_queue)
    -> batch
      -> retry (retry_on_failure)
        -> timeout
          -> sua função de push
```

Entender a ordem explica o comportamento:

O timeout é por tentativa, não pelo total. Um timeout de 30s com cinco
tentativas pode levar bem mais de 30s no pior caso.

O retry acontece dentro do consumidor da fila. Enquanto ele tenta, aquele slot
da fila está ocupado.

A fila é o que desacopla o receiver do backend. Sem ela, uma lentidão no
backend vira lentidão no cliente que enviou a telemetria.

## Configuração

```go
type Config struct {
    TimeoutConfig exporterhelper.TimeoutConfig `mapstructure:"timeout"`
    BackOffConfig configretry.BackOffConfig    `mapstructure:"retry_on_failure"`
    QueueConfig   configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
}
```

Use os construtores de default do core, não invente números:

```go
TimeoutConfig: exporterhelper.NewDefaultTimeoutConfig(),
BackOffConfig: configretry.NewDefaultBackOffConfig(),
QueueConfig:   configoptional.Default(exporterhelper.NewDefaultQueueConfig()),
```

Os campos que importam na fila:

* queue_size: tamanho máximo. A unidade depende do sizer: requests, items ou
  bytes.
* num_consumers: quantos envios simultâneos. É o seu controle de concorrência
  contra o backend.
* block_on_overflow: quando a fila enche, bloqueia esperando espaço ou devolve
  erro na hora. Bloquear aplica backpressure até o receiver, o que às vezes é o
  que você quer e às vezes derruba o cliente.
* wait_for_result: se ConsumeTraces espera o resultado do envio. Em produção
  fica false. Nos testes, true deixa tudo determinístico.
* storage: liga a fila persistente, que sobrevive a restart do processo.

## Permanente contra temporário, de novo

O day07 apresentou a ideia. Hoje o teste prova a diferença:

```go
// temporário: o helper tenta de novo
return fmt.Errorf("tentativa %d: %w", n, errBackendIndisponivel)

// permanente: uma tentativa e acabou
return consumererror.NewPermanent(errors.New("lote rejeitado"))
```

TestRetryEntregaDepoisDeFalhar mostra três falhas seguidas de uma entrega, com
quatro tentativas no total. TestErroPermanenteNaoRetenta mostra uma tentativa
só. TestDesistePassadoOPrazo mostra o que acontece quando max_elapsed_time
estoura: o dado é descartado e o erro sobe.

Regra prática para classificar: 4xx do backend, payload malformado e credencial
inválida são permanentes. 5xx, timeout, conexão recusada e rate limit são
temporários.

## Fila persistente

Sem persistência, o que estava na fila morre com o processo. Com ela, sobrevive:

```yaml
extensions:
  file_storage/queue:
    directory: /var/lib/otelcol/queue
    create_directory: true

exporters:
  otlphttp:
    sending_queue:
      storage: file_storage/queue

service:
  extensions: [file_storage/queue]
```

O custo é disco e latência de escrita. Vale para telemetria que você não pode
perder, como logs de auditoria. Para métricas de alta frequência, normalmente
não compensa.

## Limitando memória

O memory_limiter é o processor que impede o Collector de morrer por OOM. Ele
derruba dados quando a memória passa do limite, devolvendo erro temporário para
quem enviou:

```yaml
processors:
  memory_limiter:
    check_interval: 1s
    limit_percentage: 80
    spike_limit_percentage: 20
```

Ele precisa ser o primeiro da lista de processors. Depois do batch, ele derruba
dados que já custaram trabalho.

Ponto que confunde: o memory_limiter derrubar dados é sinal de que alguma coisa
antes dele está errada, seja fila grande demais, backend lento ou Collector
subdimensionado. Ele é o freio de emergência, não a solução.

## Medindo antes de otimizar

Benchmark, como no day12:

```bash
go test -bench=. -benchmem -run=XXX ./...
```

Para comparar duas versões, use benchstat e não o olho:

```bash
go test -bench=. -count=10 ./... > antes.txt
# aplique a mudança
go test -bench=. -count=10 ./... > depois.txt
benchstat antes.txt depois.txt
```

Perfil de CPU e memória a partir do teste:

```bash
go test -bench=. -cpuprofile=cpu.out -memprofile=mem.out ./...
go tool pprof -http=:8080 cpu.out
```

E no Collector em execução, com a extension pprof ligada:

```bash
go tool pprof -http=:8080 http://127.0.0.1:1777/debug/pprof/heap
go tool pprof -http=:8080 http://127.0.0.1:1777/debug/pprof/profile?seconds=30
```

O day19 traz a configuração completa com pprof.

## Onde o desempenho costuma vazar

Alocação por span. O caminho quente roda milhões de vezes. fmt.Sprintf, append
sem capacidade prévia e conversão string para bytes aparecem no topo do perfil
com frequência.

Clonagem desnecessária. Declarar MutatesData true sem precisar faz o Collector
copiar telemetria à toa em pipelines com fan-out.

Serialização repetida. Se o dado vai ser serializado, considere serializar uma
vez na entrada da fila, e não a cada tentativa de envio. É o que o kafkaexporter
faz com o caminho baseado em Request.

Log no caminho quente. Um logger.Info por span é um incêndio garantido.

## Exercícios

1. Ligue block_on_overflow, encha a fila e observe a diferença de comportamento.
2. Faça o backend falso responder devagar e veja o timeout agir.
3. Rode o benchmark do day12 com -cpuprofile e ache a função mais cara.

## Checklist do dia

* Sei a ordem das camadas do exporterhelper e o que cada uma resolve.
* Sei classificar um erro entre permanente e temporário.
* Sei quando fila persistente vale o custo.
* Sei tirar um perfil de CPU e de memória, do teste e do processo vivo.

Próximo: [Day 18, o dia a dia no monorepo](../day18/README.md)
