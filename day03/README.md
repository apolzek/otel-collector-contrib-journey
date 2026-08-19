# Day 03: Go essencial, parte 1

Objetivo do dia: a teoria de Go que você precisa para LER o código do
Collector. Nada aqui usa a API do Collector, são exemplos puros que rodam
sozinhos. O dia de hoje cobre interfaces, embedding, asserção de tipo, erros e
struct tags.

Todos os exemplos ficam em exemplos/ e rodam assim:

```bash
cd exemplos
go run ./01-interfaces
```

## 1. Interfaces são implícitas

Em Go nenhum tipo declara que implementa uma interface. Se ele tem os métodos,
ele satisfaz. Isso muda a direção da dependência: a interface é definida por
quem CONSOME, não por quem implementa.

É por isso que o Collector consegue orquestrar centenas de componentes que ele
nunca viu. Ele só conhece duas funções:

```go
type Component interface {
    Start(ctx context.Context, host Host) error
    Shutdown(ctx context.Context) error
}
```

Duas consequências práticas:

Interfaces pequenas compõem interfaces maiores. No Collector, receiver.Traces é
literalmente Component mais o método de consumo, e nada mais.

Como não existe declaração de implementação, o compilador não avisa quando você
quebra o contrato no lugar certo. Daí o idioma mais repetido do repositório:

```go
var _ component.Config = (*Config)(nil)
```

Isso é uma asserção em tempo de compilação. Custo zero em runtime, e se alguém
apagar um método o pacote deixa de compilar ali, com mensagem clara.

Uma armadilha que vale conhecer desde já: uma interface é um par (tipo, valor).
Uma interface nil e uma interface contendo um ponteiro nil são coisas
diferentes, e a segunda não é igual a nil. O exemplo 01 mostra isso.

Exemplo: exemplos/01-interfaces

## 2. Embedding é composição, não herança

Go não tem herança. O que existe é embedding: você coloca um tipo dentro de
outro sem dar nome ao campo, e os métodos dele são promovidos.

O core usa isso para dar implementações vazias de graça:

```go
type StartFunc func(context.Context, Host) error

func (f StartFunc) Start(ctx context.Context, h Host) error {
    if f == nil {
        return nil
    }
    return f(ctx, h)
}
```

Um componente que embute StartFunc e ShutdownFunc já satisfaz
component.Component sem escrever uma linha, porque o valor zero de um tipo
função é nil e o método trata isso. Você verá exatamente esse arranjo no
connector do day11.

Embutir uma interface dentro de uma struct é outro truque, muito usado em
testes: o tipo passa a satisfazer a interface imediatamente, e você sobrescreve
só o método que interessa. O preço é que chamar um método não sobrescrito entra
em pânico, porque a interface embutida é nil.

Exemplo: exemplos/02-embedding

## 3. Asserção de tipo e type switch

A forma com dois valores nunca entra em pânico:

```go
r, ok := c.(Recarregavel)
if !ok {
    return
}
```

No Collector isso vira descoberta de capacidade: pergunta-se a um valor se ele
também sabe fazer outra coisa. Dois lugares onde você vai ver isso:

* Componentes buscam extensions no host e testam se aquela extension
  implementa a interface de que precisam, por exemplo autenticação.
* As APIs experimentais com prefixo x devolvem o tipo estável, e o teste faz
  factory.(xprocessor.Factory) para acessar o que ainda não é estável.

O type switch é o irmão do assert para valores dinâmicos. Como pdata e OTTL
trabalham com any, ele é onipresente.

Exemplo: exemplos/03-assercao

## 4. Erros são valores

O Collector tem regras estritas sobre erro, e o linter reforça todas:

* Use errors.New e fmt.Errorf. As bibliotecas github.com/pkg/errors e
  hashicorp/go-multierror são proibidas pelo depguard.
* Ao embrulhar, use o verbo %w. Com %v a identidade do erro original se perde e
  errors.Is deixa de funcionar lá em cima.
* Declare erros sentinela no topo do arquivo, como var errClientNotInit =
  errors.New(...), e compare com errors.Is. Nunca compare a mensagem.
* Quando quem trata precisa dos dados do erro, e não só da identidade, crie um
  tipo de erro e use errors.As.
* Para acumular erros independentes existem errors.Join da stdlib e
  go.uber.org/multierr, este último bastante usado no repositório.

O linter errorlint reprova comparação direta com == quando o erro pode estar
embrulhado.

Exemplo: exemplos/04-erros

## 5. Struct tags e reflexão

Uma struct tag é uma string colada ao campo, lida em runtime por reflexão.
A configuração do Collector inteira depende disso:

```go
type Config struct {
    Endpoint string `mapstructure:"endpoint"`
    Timeout  time.Duration `mapstructure:"timeout"`
}
```

Três coisas que importam na prática:

* Todo campo exportado precisa de tag mapstructure. O teste
  componenttest.CheckConfigStruct reprova quem esquecer.
* A tag `mapstructure:",squash"` embute os campos do struct interno no nível de
  cima do YAML em vez de criar um bloco aninhado. É assim que configurações
  comuns, como TLS e retry, entram em dezenas de componentes.
* Campo não exportado é invisível para o unmarshal.

Aparece também o campo final `_ struct{}`, que impede alguém de construir a
config com um literal posicional. Assim, acrescentar um campo novo no meio da
struct não quebra código que já existe.

Exemplo: exemplos/05-tags

## Checklist do dia

* Sei explicar por que `var _ Interface = (*Tipo)(nil)` aparece em todo lugar.
* Sei a diferença entre embutir um tipo e embutir uma interface.
* Sei quando usar errors.Is e quando usar errors.As.
* Consigo olhar uma struct de Config e dizer como ela vira YAML.

Próximo: [Day 04, Go essencial parte 2](../day04/README.md)
