# Validação — HAME 1.1 beta

## Verificações do beta 1.1 — 2026-10-29

- A Mica executou a suíte Go, `go vet`, testes JavaScript do shim e compilação dos quatro alvos por ferramentas reais do HAME. Recursos Windows identificam `1.1.0.1` / `1.1-beta.1`.
- Testes cobrem cache persistente sem worker disponível, injeção antes dos módulos, rotas SPA e hash, sessões isoladas, redução do histórico transportado, janela real preservada na API, `/tools` no formato nativo e identidade de um GGUF fictício diferente de Gemma.
- Recuperação testada com um fato deliberadamente omitido do resumo, referências originais auditadas e isolamento de outra conversa. Entrada individual grande demais continua recusada antes da inferência.
- Teste real Windows com Gemma E4B Q4_K_M, compilador E2B Q4_K_M e llama.cpp b11946 Vulkan: compactação automática por saturação observada, com quatro mensagens originais preservadas. A API e os logs confirmaram a troca de workers.
- Navegador real: HTML e bundle obtidos do llama-server instalado, shim injetado, sete ferramentas descobertas, identidade com Q4_K_M e resposta `BETA-UI-OK` visível na conversa.
- A instalação beta fica em pasta separada, sem alterar a instalação estável nem os modelos originais. Os pacotes ARM64/Android são compilações cruzadas; não foram executados em hardware de destino.

## Verificações da 1.02 — 2026-10-29

- Reparados os textos que já estavam gravados com codificação corrompida na WebUI e no CHANGELOG. O JavaScript vendorizado foi inspecionado e suas tabelas Unicode válidas foram preservadas.
- Scripts PowerShell distribuídos em UTF-8 BOM; o inicializador usa Windows PowerShell 5.1, que não reconhece UTF-8 sem BOM automaticamente.
- Testes de regressão conferem acentos, charset HTTP, declaração UTF-8 antes dos scripts e BOM em todos os scripts PowerShell. Suíte Go e vet aprovados; quatro alvos compilados pela Mica via HAME.
- Autoteste do configurador executado no Windows PowerShell 5.1 com renderização das duas abas; acentos em memória, configuração, execução, sessão e recuperação conferidos visualmente.

## Verificações da 1.01 — 2026-10-29

- Base: checkout novo do repositório público em `e1bcd4d`, com o ciclo de ferramentas implementado no chat principal. WebUI completa e licenças preservadas.
- Windows x64, Go 1.27.3: suíte `go test ./...`, análise `go vet ./...` e compilação aprovadas. Casos novos cobrem esquema real de `/tools` da build b11946, resultados `role=tool`, histórico persistido, argumentos inválidos/aninhados, nomes não habilitados, IDs duplicados, chamadas em lote, truncamento, limite por turno, erro sem repetição automática e bloqueio de requisições de outros sites.
- Teste real com Gemma E4B Q4_K_M e llama.cpp b11946 Vulkan: o modelo solicitou `exec_shell_command`, criou um arquivo com marcador e usou `read_file` para conferir o conteúdo. O arquivo real e os registros de ambas as ferramentas foram conferidos fora do modelo.
- Elevação: o modelo executou `fltmc filters`, operação de leitura administrativa, com listagem real e `[exit code: 0]`. A mesma operação em sessão sem elevação retornou acesso negado. A configuração de execução como administrador é local; os pacotes públicos não forçam elevação.
- Compilação Windows delegada à Mica pelo próprio HAME e conferida por Gepeto: ferramenta nativa retornou exit code 0; `-version`, FileVersion e ProductVersion identificam `1.01`. Recursos de ícone incluídos.
- Compilação cruzada para Linux amd64, Linux arm64 e Android arm64 concluída. Esses binários não foram executados localmente em seus sistemas de destino nesta atualização; validação de inferência e ferramentas foi feita no Windows. A automação GitHub testa Windows e Ubuntu a cada push.
- A API `/tools` do llama.cpp é interna e experimental. O fixture em `cmd/hame/testdata/llama-b11946-tools.json` contém somente as definições públicas das sete ferramentas, sem histórico ou configuração pessoal.

## Histórico da versão 1.0 (relato preservado)

## Verificações desta atualização — 2026-10-28

- Base funcional: commit `b6531bb3283f3df9988095338108808460ecf6e4` da 0.92 RC; reorganização e identificação 1.0, sem novas funcionalidades do motor.
- WebUI: 60 arquivos recuperados do `embed.FS` do executável local 0.92 RC. Os conteúdos foram conferidos contra os hashes do próprio embed antes da atualização da identificação no HTML e no service worker. Os assets, inclusive os referenciados pelo index, estão em `cmd/hame/webui/`.
- Windows x64, Go 1.27.3: `go test ./...`, `go vet ./...` e build aprovados. O script `scripts/build-all.ps1` compilou Windows amd64, Linux amd64, Linux arm64 e Android arm64 com CGO desabilitado. Recursos Windows: FileVersion e ProductVersion `1.0`; `HAME.exe -version` retorna `HAME 1.0`.
- Configurador: autoteste de seleção de dispositivos/configuração da interface aprovado. Não houve novo probe real do llama/GPU nem nova inferência nesta atualização.
- Checkout limpo local: novo clone do commit reorganizado, sem recursos `.syso` ou saídas de build; `go test ./...`, `go vet ./...`, `go build ./cmd/hame` e `hame.exe -version` aprovados. As 60 referências locais do HTML/manifest foram conferidas; as duas licenças mantêm os blobs originais. A única alteração funcional nas fontes Go é a constante de versão.
- Especificação original fornecida preservada byte a byte em `specs/HAME_1.3.yaml`; SHA-256 `d2b67e83eee312e1411d6a08782562a101cd7bab711d3c3301555ce3d2087422`.
- Não foram gerados nem publicados novos pacotes de Release. O ZIP 1.0 da máquina de origem não foi utilizado: esta atualização foi reconstruída a partir da base 0.92 RC, conforme orientação do autor.

Linux ARM64 e Android/Termux não foram executados em hardware de destino. Os resultados WSL2 abaixo são herdados da documentação 0.92 RC, sem nova execução local. Ferramentas genéricas continuam ausentes; referências de fotos não reanalisam o original e sínteses podem omitir informações.

## Histórico de validação 0.92 RC (relato preservado)

- Windows x64: suíte Go e go vet aprovados; binário executa -version e possui FileVersion/ProductVersion 0.92 RC. Configurador aprovado com detecção assíncrona real do llama/GPU e validação, usando configuração isolada.
- Linux x64: suíte compilada para linux/amd64 e executada com sucesso no Ubuntu-24.04 WSL2. Inclui memória/compactação, histórico, fotos/YAML, relógio, API e entradas inválidas. Teste de grupos de processos confirmou morte dos descendentes próprios e sobrevivência de um processo independente. Validação manual confirmou que o executável falso não foi invocado para autodetecção. A migração automática de configuração antiga é exclusiva do Windows e foi corretamente ignorada no teste Unix.
- Linux x64: ELF estático confirmado e -version executado no Ubuntu. Configuração ausente apresenta erro; não abre configurador. Não houve inferência com um llama Linux real: esse motor não está instalado no Ubuntu usado nos testes.
- Linux ARM64: compilação cruzada concluída; ELF AArch64 estático conferido. Sem execução em hardware ARM64.
- Termux ARM64: compilação cruzada android/arm64 concluída; ELF AArch64 PIE com interpretador /system/bin/linker64 conferido. Sem execução em aparelho Android/Termux. Não confundir com o alvo linux/arm64.

Compilador instalado: Go 1.18 windows/amd64, CGO desabilitado. Os testes de inferência reais e recuperação de fotos da 0.91 estão documentados naquela versão; não são apresentados como nova validação de inferência para todos os alvos da 0.92.

A especificação HAME_1.3.yaml mantém os bytes da versão anterior. Os pacotes não incluem llama.cpp, modelos ou dados pessoais. As fontes são comuns aos quatro alvos, com arquivos específicos de controle de processos e comportamento manual Unix.

Pacote Linux x64 extraído no Ubuntu e executado com sucesso; erro de configuração ausente e testes de não sondagem/grupo de processos confirmados. Scripts Unix foram normalizados para LF e os tar.gz preservam modo 0755 para hame/start.sh. A suíte de testes Android ARM64 também compilou; não foi executada em aparelho.
