# HAME 1.02

Memória persistente sobre llama.cpp. Uma base Go para Windows x64, Linux x64, Linux ARM64 e Android ARM64 via Termux. Código próprio BSD-3-Clause; WebUI llama.cpp com licença MIT preservada. Esta versão mantém a memória por ocupação, modos Constrained/Resident, foto original + descrição YAML, histórico e relógio do sistema.

## Correções da 1.02

- Acentos e separadores corrigidos no painel da WebUI e no histórico de alterações.
- Scripts PowerShell distribuídos com UTF-8 BOM, para que o configurador mostre português corretamente no Windows PowerShell 5.1.
- Declaração UTF-8 antecipada no HTML e testes de regressão para os textos servidos e a codificação dos scripts.

## Novidades da 1.01

- **Execução real de ferramentas pelo chat:** o HAME agora descobre as ferramentas nativas do llama-server, envia suas definições ao modelo, executa chamadas estruturadas e devolve os resultados para ele continuar a tarefa.
- Leitura, busca, escrita e edição de arquivos, consulta do ambiente e comandos de shell disponíveis por configuração explícita com `tools` e `tool_cwd`. `["all"]` habilita as sete ferramentas nativas conhecidas; sem essa configuração, continuam desativadas.
- Chamadas e resultados preservados no histórico, com registro antes e depois de cada execução em `data/tool-audit`. Limite de chamadas, timeout, validação de argumentos e proteção contra requisições de outros sites.
- Ferramentas somente no chat principal; compilador de memória, extração de estado e descrição de imagens continuam sem executar ações.
- Validado no Windows com llama.cpp b11946: o modelo criou e leu um arquivo de teste e executou uma consulta administrativa de leitura (`fltmc filters`) com código de saída 0. A mesma consulta sem elevação recebeu acesso negado. O HAME não exige administrador para tarefas comuns; comandos herdam os privilégios com que ele foi iniciado.

Veja [a configuração e os limites das ferramentas](docs/API.md#ferramentas-nativas-do-llama-server). O diretório inicial não é sandbox e a API nativa do llama.cpp é experimental.

## Windows

Compile com `powershell -ExecutionPolicy Bypass -File scripts/build.ps1`. A saída fica em `build/windows-amd64/`, junto ao configurador e aos iniciadores. Execute `Configurar-HAME.cmd` nessa pasta e escolha executáveis, modelos e dispositivos. O build de recursos requer Python e clang; o HAME em si não requer esses programas para rodar. Uma configuração existente na pasta de saída é preservada.

## Linux e Termux: configuração inteiramente manual

1. Obtenha ou compile **llama.cpp para o seu próprio sistema e arquitetura**, conforme as instruções do projeto. O pacote HAME não contém llama.cpp nem GGUFs. Um executável Windows não substitui um build Linux/Android.
2. Use `configs/unix-server.json` ou `configs/unix-cli.json` como `config.json` junto ao binário. Os scripts copiam o exemplo server somente se ainda não existir uma configuração na pasta de saída. Edite os caminhos, modelos, dispositivos, camadas GPU, orçamento e capacidades de acordo com o seu llama. Caminhos relativos são resolvidos a partir da pasta do config.json; caminhos de executáveis não são procurados no PATH.
3. `main_device` e `compiler_device` são explícitos (`none` para CPU). `gpu_layers` também é explícito (`0` para CPU). O valor `auto` é recusado nesses campos. HAME não procura GPUs, executáveis nem tokenizador e não chama --help/--version/--list-devices ao iniciar. Variáveis de ambiente do usuário são preservadas.
4. Declare em `llama_capabilities` os parâmetros reais suportados, conforme o `--help` que você pode consultar manualmente no seu build. As capacidades dos exemplos incluem o protocolo mínimo necessário. Opções extras precisam constar nessa lista. Declarar um parâmetro não faz um llama incompatível passar a suportá-lo: falhas do motor são reportadas nos logs. `llama_version` é apenas uma identificação fornecida pelo usuário.
5. `llama_tokenize` é opcional e só será executado se informado explicitamente. Sem ele, CLI usa estimativa conservadora; não descobre um tokenizador ao lado do executável. Server mantém contagem via suas APIs. As chamadas normais de saúde, geração e controle de slots continuam necessárias ao funcionamento do llama.
6. Execute `chmod +x hame start.sh` e `./start.sh`, ou `./hame -config ./config.json -open=false`. Abra o endereço configurado no navegador; não há configurador nem abertura automática de navegador no Linux/Termux. `./hame -config ./config.json -check` valida a configuração declarada sem carregar os modelos. `-probe` continua disponível **somente por solicitação explícita** do operador.

Para CLI use `configs/unix-cli.json`; exige as opções nativas de saída em arquivo do llama-cli. Resident exige server com --slots, --slot-save-path, --cache-ram e --no-cache-idle-slots declarados e implementados, além da verificação real do apagamento de slots. Os exemplos CPU usam Constrained. Nem todo build futuro do llama.cpp manterá o protocolo atual: valide capacidades e teste antes de atualizar.

## Termux

Use o executável **android/arm64**, não o linux/arm64. Ambos usam ARM64, mas são alvos de sistema diferentes. Mantenha binários e scripts no diretório privado do Termux, por exemplo `$HOME/HAME`; armazenamento compartilhado como `/sdcard` não é adequado para executar binários. Configure os caminhos acessíveis ao Termux para o llama e GGUFs. Não requer proot para o HAME Android nativo. Drivers, aceleração GPU e limites de memória dependem do aparelho e do build do llama.cpp, e ficam sob controle do usuário. A execução real no Android não foi validada neste ambiente Windows/WSL.

## Compilação

O código usa apenas a biblioteca padrão do Go. Mínimo declarado: Go 1.18. Da raiz, compile com `go build ./cmd/hame`. No Linux/Termux: `sh scripts/build.sh linux amd64`, `sh scripts/build.sh linux arm64` ou `sh scripts/build.sh android arm64`. Saídas em `build/<sistema>-<arquitetura>/`, com `hame` e `start.sh`; entre nessa pasta para iniciar. Para testar nativamente: `go test ./...` e `go vet ./...`. `powershell -ExecutionPolicy Bypass -File scripts/build-all.ps1` testa o Windows, gera seus recursos e compila os quatro alvos. CGO está desabilitado nos scripts de distribuição; isso não elimina as bibliotecas exigidas pelo seu llama.cpp.

## Organização das fontes

- `cmd/hame/`: código Go e testes; `cmd/hame/webui/` contém os 60 arquivos da interface vendorizada, necessários para `//go:embed all:webui`.
- `configs/`: exemplos públicos, sem caminhos pessoais.
- `scripts/windows/` e `scripts/unix/`: configurador e iniciadores; os scripts de build ficam em `scripts/` e o gerador de recursos em `scripts/tools/`.
- `assets/`: ícone do aplicativo; `docs/`: arquitetura, API, portabilidade e validação.
- `specs/HAME_1.3.yaml`: especificação arquitetural original, preservada; a versão do aplicativo é **1.02**.
- `build/`: saídas locais ignoradas pelo Git. Modelos, llama.cpp, configurações pessoais e históricos não pertencem ao código-fonte.

A versão 1.02 corrige a codificação dos textos da 1.01, que acrescentou execução de ferramentas à base 1.0. A WebUI precisa acompanhar qualquer checkout ou arquivo de fontes; não a remova por ser um frontend já compilado.

## Processos e memória

Windows mantém Job Object para seus workers. Linux/Termux usa grupos de processos próprios, trata Ctrl+C, SIGTERM e SIGHUP e mata os grupos ao sair. Não encerra outros processos de llama. Linux também solicita SIGKILL do worker direto quando o processo/thread pai morre; um SIGKILL abrupto do HAME não oferece garantia universal de limpeza de todos os netos. Nenhum sistema garante execução após o Android forçar a parada do aplicativo.

Constrained alterna principal e compilador somente quando há compactação. Resident mantém pesos e apaga o KV/slot real, quando o backend oferece esse mecanismo. Resumos podem omitir informações; originais ficam no disco. Fotos usam server com mmproj compatível, arquivos originais e descrições YAML; até duas referências pertinentes são preservadas deterministicamente na memória compactada. Uma referência não reanalisa a foto sozinha. Data/hora do sistema é fornecida a cada turno.

Ferramentas nativas são opcionais no backend server local: configure `tools` (nomes explícitos ou `["all"]`) e `tool_cwd` absoluto no JSON. O gateway descobre as definições reais, executa chamadas estruturadas, devolve seus resultados ao principal e preserva o ciclo no histórico e em `data/tool-audit`. O compilador e a análise de imagens não executam ferramentas. São ações com os privilégios do HAME; o diretório inicial não é sandbox. Veja os limites, requisitos e detalhes da API experimental em docs/API.md. Sem `tools`, o comportamento anterior permanece.

Veja docs/API.md, docs/ARCHITECTURE.md, docs/PORTABILITY.md e docs/VALIDATION.md. A especificação original permanece em specs/HAME_1.3.yaml.
