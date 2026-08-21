# Korp — Sistema de Emissão de Notas Fiscais

Projeto técnico desenvolvido para o processo seletivo de estágio da Korp.

## Arquitetura

O sistema é composto por dois microsserviços independentes em Go, cada um com seu próprio banco de dados (SQLite), e um frontend em Angular que utiliza os dois diretamente:

- **Serviço de Estoque** (`service/estoque`, porta `8081`): dono do cadastro de produtos e do saldo. Apresenta `GET/POST /produtos`, `PUT /produtos/{codigo}` e `POST /baixas`.
- **Serviço de Faturamento** (`service/faturamento`, porta `8082`): dono das notas fiscais. Apresenta `GET/POST /notas` e `POST /notas/{id}/imprimir`. Na impressão, ele chama o Estoque via HTTP para validar e descontar o saldo.
- **Frontend** (`frontend/korp_frontend`, Angular + Angular Material, porta `4200`): duas telas — Produtos e Notas Fiscais — que utilizam os dois serviços diretamente.

Cada serviço só acessa o próprio banco. A comunicação entre Estoque e Faturamento acontece via HTTP.(Microsserviços)

## Como rodar localmente

Pré-requisitos: Go 1.26+, Node.js + Angular CLI.

Acesse `http://localhost:4200`.

## Detalhamento técnico

**Ciclos de vida do Angular utilizados:** `ngOnInit`, usado nos dois componentes de página (`ProdutosPage` e `NotasPage`) para carregar os dados assim que o componente é inicializado, em vez de no construtor.

**Uso do RxJS:** todas as chamadas HTTP retornam `Observable` (via `HttpClient`), tratadas com `.subscribe({ next, error })` para separar sucesso de falha sem quebrar a interface. O operador `finalize` é usado no fluxo de impressão de nota fiscal para garantir que o indicador de carregamento seja desligado independentemente do resultado da requisição.

**Outras bibliotecas Angular:**
- `@angular/forms` (Reactive Forms) — formulário de cadastro/edição de produto, com validações (`Validators.required`, `Validators.min`).
- Angular Signals (`signal`, `.set()`, `.update()`) — usados para o estado que muda de forma assíncrona.

**Bibliotecas de componentes visuais:** Angular Material (tema Azure/Blue).

**Gerenciamento de dependências no Golang:** Go Modules (`go.mod`/`go.sum`), nativo da linguagem — cada microsserviço tem seu próprio módulo e suas próprias dependências, reforçando o isolamento entre eles. Única dependência externa: `modernc.org/sqlite`.

**Framework utilizado no backend:** nenhum framework HTTP externo — os dois serviços usam apenas a biblioteca padrão do Go (`net/http`), incluindo o roteamento por método e wildcard (`http.ServeMux` com padrões como `"POST /notas/{id}/imprimir"`).

**Tratamento de erros e exceções no backend:**
- Middleware de *panic recovery* em ambos os serviços, evitando que um erro inesperado derrube o processo.
- Erros de negócio (validação, conflito, não encontrado) retornam JSON estruturado `{"error": "..."}` com o status HTTP apropriado (400, 404, 409).
- A chamada do Faturamento ao Estoque usa `context.WithTimeout` (5s); se o Estoque estiver fora do ar ou não responder a tempo, o Faturamento devolve `503` com uma mensagem clara, sem travar nem quebrar.

**C# / LINQ:** não aplicável — o backend foi implementado em Go.

## Requisitos obrigatórios atendidos

- [x] Arquitetura de microsserviços (Estoque + Faturamento)
- [x] Tratamento de falhas com recuperação e feedback ao usuário (Estoque fora do ar → erro 503 tratado no Faturamento e exibido na tela → recuperação ao religar o serviço)
- [x] Conexão real com banco de dados (SQLite, um banco por serviço)

## Requisitos opcionais atendidos

- **Tratamento de concorrência:** o endpoint `POST /baixas` do Estoque executa a baixa dentro de uma transação SQL, usando `UPDATE ... WHERE saldo >= quantidade`. Isso garante atomicidade mesmo com duas notas disputando o último item em estoque, sem necessidade de lock manual em código.
- **Idempotência (parcial):** o endpoint de impressão verifica se a nota já está fechada antes de chamar o Estoque; uma segunda tentativa de impressão na mesma nota não causa desconto duplicado de saldo.
