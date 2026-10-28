# Portabilidade 1.0

| Alvo | GOOS | GOARCH | Configuração |
|---|---|---|---|
| Windows x64 | windows | amd64 | GUI/JSON, capacidades sondadas pelo llama |
| Linux x64 | linux | amd64 | JSON explícito, sem sondagem automática |
| Linux ARM64 | linux | arm64 | JSON explícito, sem sondagem automática |
| Termux ARM64 | android | arm64 | JSON explícito, sem sondagem automática |

Não há suporte 32 bits nesta entrega. Linux ARM64 e Android ARM64 têm binários distintos. O HAME é compilado com CGO=0; o motor llama.cpp deve ser compatível com ABI, CPU, bibliotecas, dispositivos e modelo do destino. Os campos llama_capabilities refletem o contrato nativo exigido pelo adaptador, não são tradução de flags ou uma substituição de --help.

O protocolo server exige /health, /props, /apply-template, /tokenize e /v1/chat/completions. Resident também exige /slots e o apagamento verificável de slot. O protocolo CLI depende de --output-file e dos parâmetros declarados no exemplo. A lista de capacidades deve ser atualizada manualmente se o operador trocar de build. As capacidades não são um atestado de compatibilidade; o motor ainda precisa aceitá-las de fato.

A cópia Windows conserva o configurador. Os scripts shell não instalam dependências, não escolhem GPU, não procuram modelos e não modificam o JSON. Os exemplos são CPU, com caminhos de exemplo que precisam ser editados. As opções de orçamento ausentes ainda usam os valores de compatibilidade do núcleo; para controle completo use os exemplos preenchidos.
