# Plataforma de Simulação DCA - Go

Uma aplicação web interativa escrita em Go para simular e comparar estratégias de investimento Dollar-Cost Averaging (DCA) em Bitcoin contra Lump Sum em Ouro e S&P 500.

## Pré-requisitos

- **Go 1.16+**: Você precisa ter a linguagem Go instalada no seu sistema.
  - [Download e Instalação do Go](https://go.dev/doc/install)

## Como Rodar

1. Clone o repositório ou navegue até a pasta do projeto.
2. Execute o servidor:
   ```bash
   go run main.go
   ```
   Ou, se tiver `make` instalado:
   ```bash
   make run
   ```

3. Abra o navegador em: [http://localhost:8080](http://localhost:8080)

## Funcionalidades

- **Simulação Personalizada:** Escolha datas, valor e frequência.
- **Dados Reais:** Utiliza dados históricos do Yahoo Finance.
- **Comparação:**
  - **DCA Bitcoin:** Compras periódicas fixas.
  - **Lump Sum Bitcoin:** Compra única no início.
  - **Lump Sum Ouro:** Compra única de Ouro (XAU).
  - **Lump Sum S&P 500:** Compra única no índice americano.
- **Design Interativo:** Interface web moderna e responsiva.

## Estrutura do Projeto

- `cmd/server`: Ponto de entrada da aplicação (main.go).
- `pkg/finance`: Cliente para buscar dados históricos.
- `pkg/calculator`: Lógica de cálculo das estratégias.
- `templates`: Arquivos HTML.
- `static`: Arquivos CSS e assets estáticos.
