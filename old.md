# otel-collector-contrib-journey


---

## opentelemetry-collector (core)

O core é o **motor** do Collector: o framework onde a telemetria entra, é
processada e sai. Ele quase não traz componentes — traz as regras do jogo que
todos os componentes (inclusive os 300+ do contrib) precisam seguir.

Destrinchando cada peça:

### 1. O modelo de pipeline: receivers → processors → exporters

O pipeline é uma cadeia de consumidores. Cada peça recebe dados e passa para a
próxima; a última entrega para fora do Collector.

| tipo | papel | direção | exemplos no core |
|---|---|---|---|
| **receiver** | entrada — escuta uma porta ou vai buscar (scrape) | produz | `otlp` |
| **processor** | transforma, filtra, agrupa, limita | passa adiante | `batch`, `memory_limiter` |
| **exporter** | saída — manda para um backend | consome | `otlp`, `otlphttp`, `debug` |
| **connector** | liga a saída de um pipeline na entrada de outro | exporter de um lado, receiver do outro | `forward` |
| **extension** | capacidade que não toca em telemetria (health check, auth, zpages) | fora do pipeline | `zpages`, `memory_limiter` |

Duas dimensões que se cruzam:

- **Sinal**: `traces`, `metrics`, `logs` e `profiles` (este ainda em
  desenvolvimento). Um pipeline é *sempre* de um sinal só — não existe pipeline
  misto. Um mesmo componente pode suportar vários sinais, ou só alguns, e com
  estabilidade diferente em cada.
- **Instância**: `otlp` e `otlp/interno` são duas instâncias do mesmo tipo, com
  configs diferentes. O nome completo (`tipo/nome`) é o `component.ID`.

```yaml
receivers:
  otlp:
    protocols: { grpc: {}, http: {} }

processors:
  memory_limiter: { check_interval: 1s, limit_mib: 512 }
  batch:

exporters:
  debug: { verbosity: detailed }

service:
  pipelines:
    traces:                                  # <- pipeline do sinal traces
      receivers:  [otlp]
      processors: [memory_limiter, batch]    # <- a ordem AQUI é a ordem de execução
      exporters:  [debug]
```

Pontos que costumam pegar quem está começando:

- **A ordem dos processors importa**, e é a ordem da lista dentro do pipeline —
  não a ordem em que foram declarados no bloco `processors:`. A recomendação
  padrão é `memory_limiter` primeiro (derrubar dados antes de estourar a
  memória) e `batch` por último.
- **Declarar um componente no bloco de cima não o liga em nada.** Se ele não
  aparecer em `service::pipelines`, ele simplesmente não é instanciado.
- **Um receiver pode alimentar vários pipelines.** Quando isso acontece e algum
  processor declara `MutatesData: true`, o Collector clona os dados antes de
  distribuir — é o que impede um pipeline de corromper o outro.
- **Fan-in e fan-out são nativos**: vários receivers → vários exporters no mesmo
  pipeline, sem nada extra.

### 2. O sistema de configuração (`confmap`)

O core não usa "um parser de YAML". Ele tem uma camada própria com três etapas:

**Providers** — de onde o texto da config vem. O core traz cinco:

| provider | uso |
|---|---|
| `file` | `--config=file:/etc/otel/config.yaml` (é o default) |
| `env` | `--config=env:MINHA_CONFIG` — a config inteira numa variável |
| `yaml` | `--config="yaml:exporters::debug::verbosity: detailed"` — inline na CLI |
| `http` / `https` | busca a config de uma URL |

**Resolução** — antes do parse, resolve as referências `${...}`:

```yaml
exporters:
  otlp:
    endpoint: ${env:BACKEND_ENDPOINT}
    headers:
      authorization: ${file:/run/secrets/token}
```

E faz **merge**: você pode passar `--config` várias vezes, e os arquivos
seguintes sobrescrevem os anteriores. É assim que se separa base + overlay por
ambiente sem template engine.

**Unmarshal + Validate** — o mapa resolvido é despejado na struct `Config` de
cada componente, por cima dos defaults que a factory devolveu, seguindo as tags
`mapstructure`. Se a struct implementar `Validate() error` (interface
`confmap.Validator`), ele é chamado antes de o componente ser criado. Config
inválida = o Collector **não sobe**, com mensagem apontando o campo.

Detalhe importante: campo desconhecido no YAML é **erro**, não aviso. Um typo em
`verbosiy:` derruba a inicialização em vez de virar silêncio.

### 3. O `service`

É o orquestrador — o que roda depois que a config foi lida:

- **Monta o grafo.** Não é uma lista, é um DAG: resolve quais componentes são
  compartilhados entre pipelines, decide quem precisa clonar dados, liga os
  connectors e detecta ciclo. Se um pipeline referencia um componente que não
  existe no binário, o erro nasce aqui (`unknown type: "filelog"`).
- **Ciclo de vida.** Chama `Start()` na ordem inversa do fluxo (exporters
  primeiro, receivers por último — para o receiver nunca aceitar dado sem ter
  para onde mandar) e `Shutdown()` na ordem direta, drenando o pipeline.
- **Telemetria do próprio Collector** (`service::telemetry`): métricas e logs
  internos — `otelcol_receiver_accepted_spans`, `otelcol_exporter_send_failed_spans`,
  tamanho de fila. É o que você usa para monitorar o Collector.
- **Extensions** (`service::extensions`): sobe as extensions, que vivem fora dos
  pipelines.

### 4. `pdata`

É o modelo de dados em memória, e a fronteira que todos os componentes falam:

| pacote | sinal |
|---|---|
| `pdata/ptrace` | traces |
| `pdata/pmetric` | metrics |
| `pdata/plog` | logs |
| `pdata/pprofile` | profiles |
| `pdata/pcommon` | tipos compartilhados (`Map`, `Value`, `Timestamp`, `Resource`) |

Por que não é uma struct Go normal:

- **É gerado a partir do protobuf do OTLP.** O modelo interno é o mesmo do
  formato de fio — evita converter tudo a cada salto do pipeline.
- **A API é toda de acessores** (`rs.Resource().Attributes().PutStr(...)`), sem
  campos públicos. É isso que permite trocar a representação interna sem quebrar
  centenas de componentes.
- **É passado por referência.** Escrever num atributo altera o dado que os
  outros pipelines veem. Daí existir `consumer.Capabilities{MutatesData: true}`:
  é a declaração de "eu escrevo nos dados", e é o que faz o Collector clonar
  quando necessário.

A hierarquia se repete nos sinais: `Resource` (de quem é — `service.name`,
`host.name`) → `Scope` (qual instrumentação gerou) → o dado em si (span /
métrica / log record).

### 5. Feature gates

Mecanismo para ligar e desligar mudanças de comportamento **em tempo de deploy**,
sem precisar de outro binário. Serve para o projeto evoluir sem quebrar todo
mundo de uma vez.

```bash
otelcol --config=config.yaml --feature-gates=+minha.feature,-outra.feature
```

O ciclo de vida (copiado do Kubernetes):

| estágio | comportamento |
|---|---|
| `alpha` | **desligado** por padrão; você liga com `+` |
| `beta` | **ligado** por padrão; você desliga com `-` |
| `stable` | permanentemente ligado; tentar desligar dá erro, e o gate some numa versão futura |
| `deprecated` | permanentemente desligado; some depois de ~2 releases |

Na prática: quando o changelog diz "comportamento X mudou, controlado pelo gate
Y", é aqui que você reverte temporariamente enquanto se adapta.

### 6. OCB — OpenTelemetry Collector Builder (`cmd/builder`)

O gerador de distribuições. Você declara em YAML **quais módulos** quer, e ele
gera o código Go de um Collector e compila:

```yaml
dist:
  name: meu-otelcol
  output_path: ./meu-otelcol

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.158.0
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver v0.158.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.158.0

exporters:
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.158.0
```

```bash
builder --config=builder-config.yaml
```

O que ele gera é justamente um `components.go` — um mapa de tipo → factory,
chamando a `NewFactory()` de cada módulo listado. **Adicionar um componente ao
seu build é literalmente acrescentar uma `NewFactory` naquele mapa.** É por isso
que a factory é a peça central da arquitetura: é a única coisa que um componente
precisa expor para ser plugável.

O binário resultante contém só o que você listou — daí a recomendação de usar
OCB em produção em vez do `otelcol-contrib` inteiro (tamanho, tempo de deploy,
superfície de ataque).

### Os componentes que o core realmente traz

O conjunto mínimo para o Collector fazer algo útil sozinho:

| classe | componente | o que faz | estabilidade |
|---|---|---|---|
| receiver | `otlp` | recebe OTLP via gRPC e HTTP | stable (traces/metrics), beta (logs) |
| receiver | `nop` | não faz nada; existe para testes | stable |
| processor | `batch` | agrupa em lotes antes de exportar | beta |
| processor | `memory_limiter` | derruba dados quando a memória passa do limite | beta |
| processor | `queuebatch` | sucessor do `batch`, reusa a fila do `exporterhelper` | development |
| exporter | `otlp` | envia OTLP via gRPC | stable (traces/metrics), beta (logs) |
| exporter | `otlphttp` | envia OTLP via HTTP | stable (traces/metrics), beta (logs) |
| exporter | `debug` | imprime a telemetria no stdout — ferramenta nº 1 de troubleshooting | development |
| exporter | `nop` | descarta tudo; para testes e benchmark | stable |
| extension | `zpages` | páginas de diagnóstico ao vivo (`/debug/tracez`, `/debug/pipelinez`) | beta |
| extension | `memory_limiter` | limite de memória compartilhado entre componentes | development |
| connector | `forward` | encadeia a saída de um pipeline na entrada de outro | beta |

Fora isso, o core carrega os **helpers** que os componentes de terceiros usam
para não reimplementar o difícil: `exporterhelper` (fila persistente, retry com
backoff, timeout), `processorhelper`, `receiverhelper`, `scraper` (esqueleto de
coleta por polling) e os pacotes `*test` (`consumertest`, `processortest`).

### Uma correção sobre o binário `otelcol`

O repositório core **não publica binários**. O que ele tem é:

- o pacote `otelcol/`, que é a *biblioteca* para montar um Collector
  (`otelcol.NewCommand`, resolução de config, ciclo de vida);
- `cmd/otelcorecol/`, um build de desenvolvimento gerado pelo próprio OCB a
  partir do `distributions.yaml` — usado para testar o core, não para produção.

A distribuição chamada `otelcol` que você baixa (`.deb`, `.rpm`, imagem Docker)
é montada e publicada por
[opentelemetry-collector-releases](https://github.com/open-telemetry/opentelemetry-collector-releases),
que mantém cinco distribuições oficiais — incluindo uma curada de propósito
geral, que junta o core com um punhado selecionado de componentes do contrib
(Prometheus, Kafka, Jaeger, Zipkin, hostmetrics).

### Manutenção e versionamento

O core é mantido diretamente pelos maintainers do projeto OTel, com barra de
revisão e estabilidade mais alta, e segue o
[VERSIONING.md](https://github.com/open-telemetry/opentelemetry-collector/blob/main/VERSIONING.md):
as APIs estáveis vão para o módulo `v1.x` (`component`, `pdata`, `consumer`,
`confmap`) e o que ainda pode mudar fica em `v0.x`. É por isso que o `go.mod` de
um componente mistura as duas numerações — `component v1.64.0` e
`processorhelper v0.158.0` são a **mesma release**.

---

## Neste repositório

- [`factory/`](factory) — o padrão Factory, do conceito puro (Go sem
  dependências) até um processor real escrito com a API do Collector.

---

mkdir -p exporter/meuexporter && cd exporter/meuexporter
go mod init github.com/open-telemetry/opentelemetry-collector-contrib/exporter/meuexporter

metadata.yaml

```
type: meu
display_name: Meu Exporter

status:
  class: exporter
  stability:
    development: [traces]     # componente novo SEMPRE começa em development
  distributions: [contrib]
  codeowners:
    active: [seu-user, seu-sponsor, terceiro]
Abra a issue Component donation e espere um sponsor (approver/maintainer) aceitar. Sem isso, os passos 12+ não vão a lugar nenhum. Precisa de 3 codeowners.

Se é exporter interno seu, pule direto pro passo 1 e no fim use OCB (lição 17).

---
1. Nome e diretório

Convenção: pasta exporter/<nome>exporter, type sem o sufixo.

mkdir -p exporter/meuexporter && cd exporter/meuexporter

2. go.mod — cada componente é um módulo próprio

go mod init github.com/open-telemetry/opentelemetry-collector-contrib/exporter/meuexporter

Ajuste para go 1.25.0 e crie o Makefile de uma linha:

include ../../Makefile.Common

3. metadata.yaml — a fonte da verdade

type: meu
display_name: Meu Exporter

status:
  class: exporter
  stability:
    development: [traces]     # componente novo SEMPRE começa em development
  distributions: [contrib]
  codeowners:
    active: [seu-user, seu-sponsor, terceiro]

tests:
  config:
    endpoint: "http://localhost:1234"
    api_key: "test"
```

### 6. **Você nunca implementa a interface `Factory` na mão — sempre `xxx.NewFactory(...)`.**

```go
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType("tutorialredact"),
		createDefaultConfig,
		processor.WithTraces(createTraces, component.StabilityLevelDevelopment),
	)
}