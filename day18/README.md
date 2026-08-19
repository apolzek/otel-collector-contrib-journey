# Day 18: o dia a dia no monorepo e o seu primeiro PR

Objetivo do dia: trabalhar dentro do contrib sem tropeçar na estrutura, e fazer
uma primeira contribuição que não é um componente novo. A maioria das
contribuições aceitas não é.

```bash
cd exemplos
go run .
```

## Preparando o ambiente

```bash
git clone https://github.com/SEU-USUARIO/opentelemetry-collector-contrib.git
cd opentelemetry-collector-contrib
git remote add upstream https://github.com/open-telemetry/opentelemetry-collector-contrib.git
make install-tools
```

O make install-tools baixa as ferramentas do repositório: mdatagen, chloggen,
crosslink, multimod, o linter. Sem elas a maioria dos alvos falha.

## Os comandos que você vai usar

Trabalhe sempre dentro do diretório do componente. Rodar make na raiz percorre
mais de trezentos módulos e demora muito:

```bash
cd receiver/filelogreceiver
make test
make lint
make generate
```

Da raiz, os que valem conhecer:

| comando | o que faz |
|---|---|
| make otelcontribcol | compila o binário com todos os componentes, em ./bin |
| make gotest | roda os testes de tudo, demorado |
| make golint | roda o linter em tudo, demorado |
| make crosslink | insere as diretivas replace entre módulos do repositório |
| make gotidy | roda go mod tidy em todos os módulos |
| make chlog-new | cria a entrada de changelog |
| make chlog-validate | valida a entrada de changelog |
| make generate | roda o mdatagen onde houver metadata.yaml |
| make gencodeowners | regenera o arquivo CODEOWNERS |

Para testar sua mudança de verdade:

```bash
make otelcontribcol
./bin/otelcontribcol_linux_amd64 --config minha-config.yaml
```

## Módulos locais: crosslink e go.work

Cada componente é um módulo separado, então um componente que depende de outro
precisa de uma diretiva replace apontando o caminho no disco. No contrib isso é
automático:

```bash
make crosslink
```

Fora do monorepo, ou para ligar módulos temporariamente sem sujar os go.mod,
existe o go.work. O diretório exemplos deste dia tem um funcionando:

```
go 1.25.0

use (
	.
	../../day07/printexporter
	../../day09/tagprocessor
)
```

Com ele, o programa em exemplos importa dois componentes de módulos vizinhos
sem nenhum replace, e imprime o que cada factory declara:

```
exporter: print
  config padrão: &{Path:stdout Prefix: _:{}}
  traces   Development
  metrics  Undefined
  logs     Development
```

Repare que metrics aparece como Undefined: é assim que uma factory diz que não
suporta aquele sinal.

O go.work vale só para comandos rodados a partir do diretório onde ele está.
No seu fork do contrib, não commite go.work: ele é ferramenta local, e o
projeto usa crosslink.

## Escolhendo o que fazer

Os melhores pontos de entrada, em ordem de esforço:

Issues com o rótulo good first issue e help wanted. Comece por elas, existem
centenas.

Documentação de componente. Muito README está desatualizado em relação à
configuração real. É contribuição valiosa e de revisão rápida.

Testes faltando. Achar um caminho de erro sem cobertura e escrever o teste é
bem-vindo em qualquer componente.

Adicionar uma métrica a um receiver existente. O caminho é o do day16: editar o
metadata.yaml, rodar go generate ./..., preencher o Record na coleta e testar.
Este é um dos fluxos mais comuns do repositório.

Corrigir um bug reportado. Reproduza com um teste que falha, corrija, e deixe o
teste no PR. Um PR assim é muito mais fácil de aceitar.

Componente novo é a contribuição mais difícil de todas, porque depende de achar
um sponsor. Está no day14, e não é por onde começar.

## Comandos por comentário

O repositório automatiza a triagem por comentários na issue ou no PR:

```
/label receiver/prometheus help-wanted -exporter/prometheus
/rerun
```

O primeiro adiciona e remove rótulos, e o hífen na frente remove. O segundo
roda de novo os workflows que falharam no último commit do seu PR. Existe
também /workflow-approve, usado por triagers e mantenedores para liberar os
workflows de quem contribui de fora.

## O CI

O build-and-test detecta o escopo a partir do diff: se você mexeu num
componente só, ele roda os testes daquele grupo, não do repositório inteiro.
Dá para prever localmente:

```bash
bash .github/workflows/scripts/compute-ci-scope.sh
```

O rótulo ci:full força a matriz completa.

## O fluxo de um PR

```bash
git checkout -b corrige-parse-do-filelog
# mude o código e escreva o teste
cd receiver/filelogreceiver && make lint && make test && make generate
cd ../.. && make chlog-new
# preencha o .chloggen/<sua-branch>.yaml
make chlog-validate
git add -A && git commit -m "[receiver/filelog] corrige parse de data ISO"
git push origin corrige-parse-do-filelog
```

Antes de pedir revisão, confira que o git está limpo depois do generate:

```bash
git diff --exit-code
```

Se sujar, é porque você editou à mão algo que é gerado, ou esqueceu de
commitar o resultado.

Durante a revisão, acrescente commits em vez de refazer o histórico. Force push
deixa comentários órfãos e faz o revisor perder o fio. O merge é squash, então
seus commits intermediários somem no final de qualquer jeito.

## Quando travar

Canal #otel-collector-dev no Slack da CNCF, para dúvidas de desenvolvimento.
Canal #otel-collector, para dúvidas de uso. As reuniões do SIG do Collector
são abertas e ficam no calendário público do OpenTelemetry.

Perguntar cedo economiza semanas, principalmente quando a dúvida é se uma ideia
tem chance de ser aceita.

## Checklist do dia

* Sei rodar teste, lint e generate no escopo certo, sem esperar o repo inteiro.
* Sei ligar módulos locais com go.work e sei que o repo usa crosslink.
* Tenho pelo menos três tipos de primeira contribuição em mente.
* Sei o fluxo completo de um PR, do branch ao changelog.

Próximo: [Day 19, operação, troubleshooting e segurança](../day19/README.md)
