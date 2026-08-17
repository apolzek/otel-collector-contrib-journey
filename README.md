# otel-collector-contrib-journey

Notas de estudo sobre o **OpenTelemetry Collector** e o repositório
[opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib):
como o projeto é organizado, quais interfaces e padrões de Go você precisa
dominar e o que é preciso para contribuir com um componente novo.

## Índice

| # | Documento | Sobre |
|---|---|---|
| 1 | [Guia técnico de contribuição](1.md) | Repos oficiais, aspectos fundamentais do monorepo, contagem de componentes e pacotes. |
| 2 | [Arquitetura do core](2.md) | _(pendente)_ Pipeline, `confmap`, `service`, `pdata`, feature gates e OCB. |
| 3 | [Interfaces principais](3.md) | `component.Component`, `component.Host` e as marker interfaces que todo componente implementa. |
| 4 | [Conceitos de Go que aparecem o tempo todo](4.md) | Factory + functional options, prefixo `x` (experimental), erros, concorrência, generics no OTTL, testes table-driven. |
| 5 | [Estudo de caso: `exporter/kafkaexporter`](5.md) | Arquivo por arquivo de um exporter real — código, metadados gerados, testes e `internal/`. |

## Outros arquivos

- [`old.md`](old.md) — rascunho anterior e mais longo, com o core destrinchado
  peça por peça e o passo a passo de criação de um componente.
- [`tips`](tips) — lembretes soltos sobre o processo de doação de componente.

## Ordem sugerida de leitura

1. Comece pelo [1](1.md) para entender o tamanho e o formato do monorepo.
2. Siga para [3](3.md) e [4](4.md): são as interfaces e os idiomas de Go que
   você vai reencontrar em qualquer componente.
3. Feche com [5](5.md), que aplica tudo isso em um componente que existe de
   verdade no contrib.

## Licença

[MIT](LICENSE)
