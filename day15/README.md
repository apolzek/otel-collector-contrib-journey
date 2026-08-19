# Day 15: OTTL, a linguagem de transformação

Objetivo do dia: entender o OTTL e escrever uma função nova para ele. É o
componente do contrib com mais alcance: quem domina OTTL resolve metade dos
pedidos de componente novo sem escrever componente nenhum.

```bash
cd exemplos
go run ./01-executando
go run ./02-funcao-customizada
go test ./...
```

## Por que isto importa

O OTTL vive em pkg/ottl e é usado pelo transformprocessor, pelo
filterprocessor, pelo routingconnector e por vários outros. Quando alguém abre
uma issue pedindo um processor que apaga um atributo, renomeia uma métrica ou
descarta spans de health check, a resposta correta quase sempre é OTTL.

Por isso o day09 termina com aquele aviso: antes de propor um componente,
verifique se o OTTL já resolve. Hoje você entende o que ele consegue fazer.

## A forma de um statement

```
set(span.attributes["env"], "prod") where span.attributes["duracao.ms"] > 500
```

Três partes:

Um editor, que é a função que altera algo. Aqui é set. Editores só existem no
começo do statement.

Os argumentos, que podem ser paths, literais, listas, mapas, expressões
matemáticas ou converters.

Um where opcional, que transforma o statement em condicional. Sem where, o
statement roda para todo dado que passar.

Editores e converters são coisas diferentes. Editor muda o dado e não devolve
valor útil, como set, delete_key, keep_matching_keys. Converter calcula e
devolve um valor, e por convenção começa com letra maiúscula, como ConvertCase,
Concat, Split, Hash. Converter nunca altera nada.

## Contextos

O OTTL não tem um único mundo, tem um por tipo de dado. Cada contexto define
quais paths existem:

| contexto | dado |
|---|---|
| ottlresource | Resource |
| ottlscope | Scope |
| ottlspan | span |
| ottlspanevent | evento dentro de um span |
| ottlmetric | métrica |
| ottldatapoint | ponto de dados |
| ottllog | log record |
| ottlprofile | profile |

O contexto define o que você alcança. Dentro de ottlspan você acessa span.name,
span.attributes, e também resource.attributes e scope.name, porque eles estão
acima na hierarquia. O contrário não vale: de ottlresource você não enxerga
spans individuais.

No YAML do transformprocessor isso aparece assim:

```yaml
processors:
  transform:
    error_mode: ignore
    trace_statements:
      - set(resource.attributes["env"], "prod")
      - delete_key(span.attributes, "http.request.header.authorization")
    metric_statements:
      - set(metric.description, "") where metric.name == "temp"
```

O error_mode decide o que acontece quando um statement falha em runtime:
ignore segue em frente, silent nem loga, propagate devolve o erro ao pipeline.
Em produção, ignore é o padrão sensato.

## Como o OTTL executa

Isto explica o custo e os erros:

O parse acontece uma vez, na construção do componente. Erro de sintaxe impede
o Collector de subir, que é o comportamento desejado: melhor falhar no deploy
do que descobrir em produção.

A execução acontece uma vez por dado. Um statement num pipeline com cem mil
spans por segundo roda cem mil vezes por segundo. Tudo que puder ser resolvido
no parse deve ficar fora da execução.

Exemplo: exemplos/01-executando percorre esses dois momentos com statements
reais.

## Escrevendo uma função nova

Quatro passos, e exemplos/02-funcao-customizada implementa os quatro.

Passo 1, a struct de argumentos. Os nomes dos campos viram os nomes dos
argumentos nomeados, em snake_case, e a ordem vira a ordem posicional. Os tipos
dizem ao parser o que aceitar:

```go
type MascararArguments[K any] struct {
    Target  ottl.PMapGetSetter[K]
    Key     ottl.StringGetter[K]
    Visivel ottl.Optional[int64]
}
```

Repare no K. É o mesmo parâmetro de tipo do day04: a função é escrita uma vez e
serve para qualquer contexto.

Passo 2, a factory, cujo nome é o nome usado no YAML:

```go
func NewMascararFactory[K any]() ottl.Factory[K] {
    return ottl.NewFactory("mascarar", &MascararArguments[K]{}, criarMascarar[K])
}
```

Passo 3, a implementação, que devolve uma closure executada por dado. O default
de um argumento opcional é resolvido fora da closure, uma vez só.

Passo 4, registrar no mapa de funções entregue ao parser:

```go
funcs := ottlfuncs.StandardFuncs[*ottlspan.TransformContext]()
f := NewMascararFactory[*ottlspan.TransformContext]()
funcs[f.Name()] = f
```

Chamadas possíveis depois disso:

```
mascarar(span.attributes, "cartao")
mascarar(span.attributes, "cartao", 8)
mascarar(span.attributes, "cartao", visivel=0)
```

Argumento nomeado usa igual, e permite pular opcionais anteriores.

## Erros comuns

Chave ausente não deve ser erro. No exemplo, mascarar numa chave que não existe
simplesmente não faz nada. Devolver erro aí encheria o log a cada span.

Não faça trabalho caro dentro da closure. Compilar regex, abrir conexão ou
alocar buffer na execução multiplica o custo por cada dado.

Editor devolve nil como valor. Quem devolve valor é converter.

## Se você for contribuir com uma função

O pkg/ottl tem um CONTRIBUTING.md próprio. O que ele pede, resumido: a função
precisa ser genérica sobre K, ter testes cobrindo os tipos de entrada, estar
documentada no README do pacote ottlfuncs e não duplicar algo que já existe
combinando funções.

## Exercícios

1. Escreva um converter Dominio que extrai o domínio de um email.
2. Faça mascarar funcionar também no contexto de log, sem mudar a
   implementação. Se ela estiver correta, é só trocar o tipo no registro.
3. Reescreva o tagprocessor do day09 como statements de transformprocessor e
   compare o esforço.

## Checklist do dia

* Sei a diferença entre editor e converter.
* Sei qual contexto usar e o que cada um alcança.
* Sei que o parse é uma vez e a execução é por dado.
* Consigo escrever, registrar e testar uma função OTTL nova.

Próximo: [Day 16, mdatagen](../day16/README.md)
