# Validação — HAME 1.0

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
