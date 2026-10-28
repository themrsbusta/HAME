# HAME 0.92 RC

Memória persistente sobre llama.cpp. Uma base Go para Windows x64, Linux x64, Linux ARM64 e Android ARM64 via Termux. Código próprio BSD-3-Clause; WebUI llama.cpp com licença MIT preservada. Esta versão mantém a memória por ocupação, modos Constrained/Resident, foto original + descrição YAML, histórico e relógio do sistema.

## Windows

Compile com `powershell -ExecutionPolicy Bypass -File build.ps1`. O configurador WinForms e a detecção do llama/GPU continuam disponíveis. Execute Configurar-HAME.cmd e escolha executáveis, modelos e dispositivos. O build de recursos requer Python e clang; o HAME em si não requer esses programas para rodar.

## Linux e Termux: configuração inteiramente manual

1. Obtenha ou compile **llama.cpp para o seu próprio sistema e arquitetura**, conforme as instruções do projeto. O pacote HAME não contém llama.cpp nem GGUFs. Um executável Windows não substitui um build Linux/Android.
2. Copie o exemplo correspondente de `configs/` para `config.json`. Edite os caminhos, modelos, dispositivos, camadas GPU, orçamento e capacidades de acordo com o seu llama. Caminhos relativos são resolvidos a partir da pasta do config.json; caminhos de executáveis não são procurados no PATH.
3. `main_device` e `compiler_device` são explícitos (`none` para CPU). `gpu_layers` também é explícito (`0` para CPU). O valor `auto` é recusado nesses campos. HAME não procura GPUs, executáveis nem tokenizador e não chama --help/--version/--list-devices ao iniciar. Variáveis de ambiente do usuário são preservadas.
4. Declare em `llama_capabilities` os parâmetros reais suportados, conforme o `--help` que você pode consultar manualmente no seu build. As capacidades dos exemplos incluem o protocolo mínimo necessário. Opções extras precisam constar nessa lista. Declarar um parâmetro não faz um llama incompatível passar a suportá-lo: falhas do motor são reportadas nos logs. `llama_version` é apenas uma identificação fornecida pelo usuário.
5. `llama_tokenize` é opcional e só será executado se informado explicitamente. Sem ele, CLI usa estimativa conservadora; não descobre um tokenizador ao lado do executável. Server mantém contagem via suas APIs. As chamadas normais de saúde, geração e controle de slots continuam necessárias ao funcionamento do llama.
6. Execute `chmod +x hame start.sh` e `./start.sh`, ou `./hame -config ./config.json -open=false`. Abra o endereço configurado no navegador; não há configurador nem abertura automática de navegador no Linux/Termux. `./hame -config ./config.json -check` valida a configuração declarada sem carregar os modelos. `-probe` continua disponível **somente por solicitação explícita** do operador.

Para CLI use `configs/unix-cli.json`; exige as opções nativas de saída em arquivo do llama-cli. Resident exige server com --slots, --slot-save-path, --cache-ram e --no-cache-idle-slots declarados e implementados, além da verificação real do apagamento de slots. Os exemplos CPU usam Constrained. Nem todo build futuro do llama.cpp manterá o protocolo atual: valide capacidades e teste antes de atualizar.

## Termux

Use o executável **android/arm64**, não o linux/arm64. Ambos usam ARM64, mas são alvos de sistema diferentes. Mantenha binários e scripts no diretório privado do Termux, por exemplo `$HOME/HAME`; armazenamento compartilhado como `/sdcard` não é adequado para executar binários. Configure os caminhos acessíveis ao Termux para o llama e GGUFs. Não requer proot para o HAME Android nativo. Drivers, aceleração GPU e limites de memória dependem do aparelho e do build do llama.cpp, e ficam sob controle do usuário. A execução real no Android não foi validada neste ambiente Windows/WSL.

## Compilação

O código usa apenas a biblioteca padrão do Go. Mínimo declarado: Go 1.18. No Linux/Termux com Go instalado: `sh build.sh linux amd64`, `sh build.sh linux arm64` ou `sh build.sh android arm64`. Saídas em `build/`. Para testar nativamente: `go test ./...` e `go vet ./...`. `build-all.ps1` gera os quatro alvos por compilação cruzada no Windows; execute primeiro build.ps1 para embutir os recursos Windows. CGO está desabilitado no HAME; isso não elimina as bibliotecas exigidas pelo seu llama.cpp.

## Processos e memória

Windows mantém Job Object para seus workers. Linux/Termux usa grupos de processos próprios, trata Ctrl+C, SIGTERM e SIGHUP e mata os grupos ao sair. Não encerra outros processos de llama. Linux também solicita SIGKILL do worker direto quando o processo/thread pai morre; um SIGKILL abrupto do HAME não oferece garantia universal de limpeza de todos os netos. Nenhum sistema garante execução após o Android forçar a parada do aplicativo.

Constrained alterna principal e compilador somente quando há compactação. Resident mantém pesos e apaga o KV/slot real, quando o backend oferece esse mecanismo. Resumos podem omitir informações; originais ficam no disco. Fotos usam server com mmproj compatível, arquivos originais e descrições YAML; até duas referências pertinentes são preservadas deterministicamente na memória compactada. Uma referência não reanalisa a foto sozinha. Ferramentas genéricas não são executadas pelo gateway; data/hora do sistema é fornecida a cada turno.

Veja docs/API.md, docs/ARCHITECTURE.md, docs/PORTABILITY.md e docs/VALIDATION.md. A especificação original permanece em specs/HAME_1.3.yaml.
