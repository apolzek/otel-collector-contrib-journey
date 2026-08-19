# Day 13: as boas práticas que os mantenedores cobram

Objetivo do dia: condensar o que aparece repetidamente nas revisões de PR do
Collector. Nada aqui é opinião pessoal, é o que está escrito nos guias do
projeto e o que os revisores pedem na prática.

## Antes de escrever qualquer código

Pergunte se o componente precisa existir. É a primeira pergunta que um
mantenedor faz, e ela para muita gente:

* Um componente existente já resolve? Consulte o registry do OpenTelemetry.
* O transformprocessor ou o filterprocessor, com OTTL, resolvem sem código novo?
* Se existe implementação concorrente, os autores dela foram convidados a
  colaborar numa solução comum?

Segunda pergunta: o componente precisa estar no contrib? Não precisa. O
Collector é plugável, e você pode publicar seu componente no seu próprio
repositório como um módulo Go, registrá-lo no registry e usá-lo com o OCB. O
projeto inclusive recomenda começar assim, para ganhar uso e feedback antes de
propor a doação.

## Configuração

A superfície de configuração é o contrato mais caro de manter. Uma vez publicada
em alpha, mudar significa quebrar usuários.

* Reuse as configs comuns do core. confighttp.ClientConfig,
  configgrpc.ClientConfig, configtls, configretry.BackOffConfig,
  exporterhelper.QueueBatchConfig, configopaque.String para segredos. Elas
  trazem nomes que o usuário já conhece, mais autenticação por extension e
  telemetria padronizada.
* Nomes em snake_case minúsculo, alinhados com componentes que já existem. Se
  outro exporter chama de sending_queue, chame de sending_queue.
* Valide no boot, não no primeiro dado. Erro de configuração precisa impedir o
  Collector de subir.
* Segredos nunca em string comum. Use configopaque.String, que impede o valor de
  vazar em log e em dump de configuração.
* Não invente batching, retry ou worker pool dentro do seu receiver ou exporter.
  Isso é trabalho dos helpers e dos processors, que já foram testados e são
  reutilizáveis.

## Código

* Nunca implemente a interface Factory na mão. Sempre o construtor do core.
* Use os helpers: exporterhelper, processorhelper, scraper. Um exporter escrito
  sem exporterhelper chega na revisão com pedido para reescrever.
* context.Context como primeiro parâmetro, sempre. Respeite o cancelamento.
* Erros com errors.New e fmt.Errorf com %w. As bibliotecas pkg/errors e
  hashicorp/go-multierror são bloqueadas pelo depguard.
* Log estruturado com zap, vindo de set.Logger. Nunca fmt.Println.
* Nada de log por span. Um erro logado no caminho quente vira gigabytes por hora.
  Agregue, ou use o logger com amostragem.
* Nenhuma goroutine sobrevive ao Shutdown.
* Código portátil. Use path/filepath para caminhos, nada de barra invertida
  fixa nem caminho absoluto de uma plataforma só.
* Semantic conventions para nomes de métricas e atributos. Não invente
  http.status se já existe http.response.status_code.

## Estabilidade

Todo componente novo começa em development. A progressão é
development, alpha, beta, stable, e cada degrau tem exigências crescentes de
compatibilidade, teste e documentação.

O que isso significa na prática:

* Em development e alpha, você ainda pode mudar configuração, avisando no
  changelog.
* A partir de beta, mudança de configuração exige ciclo de depreciação.
* A estabilidade é por sinal. Um receiver pode estar em beta para metrics e em
  development para logs, e isso fica registrado no metadata.yaml.

## metadata.yaml e código gerado

O metadata.yaml é a fonte da verdade do componente: tipo, classe, estabilidade
por sinal, codeowners, distribuições e a telemetria interna.

A partir dele, o mdatagen gera o cabeçalho do README, o documentation.md, o
pacote internal/metadata e os testes generated_component_test.go e
generated_package_test.go.

Regra prática: se o arquivo tem generated_ no nome, ou é o documentation.md, ele
sai de make generate. Editar à mão é rejeitado na revisão, e o CI detecta,
porque roda make generate e falha se o git ficar sujo.

## Changelog

Toda mudança que afeta usuário precisa de uma entrada:

```bash
make chlog-new       # cria .chloggen/<sua-branch>.yaml
make chlog-validate
```

O arquivo tem change_type (breaking, deprecation, new_component, enhancement,
bug_fix), component, note, issues e change_logs. Existem dois changelogs: o de
usuário e o de API, para quem importa os pacotes como biblioteca.

Se a mudança não afeta usuário nem API exportada, comece o título do PR com
[chore] ou peça o rótulo Skip Changelog.

## Pull request

Título no formato que o repositório espera:

```
[processor/tailsampling] fix AND policy
```

Ou seja, classe e nome do componente entre colchetes, depois uma frase curta.

Outras convenções:

* Um PR faz uma coisa. PR grande demora para revisar e cansa o revisor.
* Se o PR fecha uma issue, use Resolves, Fixes ou Closes no corpo.
* Marque os codeowners do componente. Eles são quem revisa de fato.
* Rode make lint e make test no diretório do componente antes de abrir.
* Se você usou uma ferramenta de IA para ajudar, a descrição do PR e os
  comentários continuam sendo sua responsabilidade, escritos por você. O
  repositório tem regra explícita sobre isso, no AGENTS.md, e uma caixa no
  template que só você pode marcar.

## Revisão

Duas coisas ajudam mais do que qualquer outra:

Responda rápido e sem defensividade. O revisor não está atacando seu código,
ele vai ser responsável por manter aquilo por anos.

Não force push depois que a revisão começou. Comentários ficam órfãos e o
revisor perde o fio. Acrescente commits, o merge é squash de qualquer jeito.

## Depois do merge

Ser code owner é responsabilidade contínua: triagem de issues, revisão de PRs e
atualização quando as interfaces instáveis do core mudarem.

Componente sem manutenção é marcado como unmaintained e sai da distribuição
padrão. Componente que quebra o build ou tem teste falhando é removido. Não é
ameaça, é o que mantém trezentos componentes num repositório só.

## Checklist do dia

* Sei justificar por que meu componente precisa existir e por que no contrib.
* Reuso as configs e os helpers do core em vez de reinventar.
* Sei que arquivo gerado não se edita à mão.
* Sei abrir um PR no formato esperado, com changelog.

Próximo: [Day 14, do zero ao hero](../day14/README.md)
