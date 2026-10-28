# HAME 0.9 RC

<p align="center"><img src="assets/HAME.png" alt="HAME: computador pixel art com cérebro na tela" width="180"></p>

**Memória persistente sobre llama.cpp: histórico completo, compilação automática de contexto e descarte do KV cache após cada resposta.**

HAME (Hierarchical Async Memory Engine) coordena um modelo principal e um compilador de memória leve. Esta versão para **Windows x64** inclui a interface oficial do llama-server, um configurador gráfico e uma janela de console que controla o ciclo de vida dos workers. Código próprio sob **[BSD-3-Clause](LICENSE)**; autor da especificação: **Busta Supastar**.

A versão do aplicativo é **0.9 RC**. A especificação arquitetural original é HAME 1.3 e está preservada em [specs/HAME_1.3.yaml](specs/HAME_1.3.yaml); são identificadores distintos.

## Começar

1. Extraia o pacote HAME-0.9-RC-windows-x64.zip, ou compile este repositório.
2. Execute **Configurar-HAME.cmd**. Na primeira execução sem configuração válida, o configurador abre automaticamente.
3. Escolha o motor, seu executável e os dois modelos GGUF.
4. Clique em **Detectar GPUs / versão** e selecione os dispositivos desejados.
5. Clique em **Salvar e iniciar**. Nas próximas vezes, use **Iniciar-HAME.cmd**.

A conversa abre no navegador; o console permanece aberto. **Fechar o console ou pressionar Ctrl+C encerra o HAME e os workers que ele criou.** Não existe mais ícone na área de notificações. Fechar somente a aba do navegador mantém o motor ativo.

O pacote não inclui llama.cpp, suas DLLs ou modelos GGUF. Escolha uma distribuição completa do llama.cpp e mantenha suas DLLs junto do executável correspondente. O configurador não baixa nem substitui esses arquivos.

## Seleção do motor

| Motor | Integração | Preparação da próxima mensagem |
| --- | --- | --- |
| **llama-server** | HTTP/JSON; o principal local reinicia após cada resposta | Compilador gera o pacote; o contexto é pré-carregado no novo principal com n_predict=0 |
| **llama-cli** | Um processo por geração; saída lida do arquivo do próprio CLI | Pacote compilado salvo; o principal carrega essa memória quando a próxima mensagem chega |

Server é o padrão recomendado para a integração com interface web e pré-carregamento. CLI é uma alternativa selecionável para quem prefere execução por tarefa. Ele recarrega o modelo a cada chamada e não mantém prefill residente. O modo resident está disponível somente para server; nele o compilador pode permanecer carregado, mas o principal local ainda reinicia para descartar seu cache antigo.

O HAME inspeciona --version e --help para verificar as capacidades necessárias. Parâmetros opcionais só são usados quando anunciados. O adaptador CLI exige a saída em arquivo e valida seu formato; não interpreta banners ou a ecoagem truncada do terminal. Uma mudança incompatível gera diagnóstico e exige reconfiguração ou uso de outra versão. **Não há garantia de compatibilidade com toda atualização futura do llama.cpp.**

## GPUs e CPU

O configurador executa --list-devices no executável selecionado e lista os IDs, nomes e memória informados por ele. Não presume Vulkan0, fabricante, índice ou quantidade de VRAM. Se a identidade de uma GPU configurada mudar, solicita nova seleção.

Pode escolher uma GPU diferente para cada worker, Automático ou Somente CPU. Vulkan suporta GPUs com driver Vulkan compatível; CPU utiliza o backend de CPU do llama.cpp. Distribuições CUDA, SYCL ou outros backends podem anunciar seus próprios dispositivos; apenas o Vulkan instalado foi validado com hardware real nesta revisão. A memória anunciada para uma GPU integrada pode ser compartilhada, não VRAM dedicada.

## Ciclo de memória

1. O principal recebe a memória interna compilada e a nova mensagem.
2. Pergunta e resposta são salvas juntas em data/context/<sessão>/.
3. O processo principal local é encerrado; seu KV cache antigo deixa de existir.
4. A resposta chega à interface enquanto o compilador prepara a memória em segundo plano.
5. No server, um novo principal recebe prefill; no CLI, a memória fica pronta no disco para a próxima chamada.

O histórico completo permanece visível na interface e preservado no disco. O gateway não repassa o histórico bruto que a interface envia ao principal. A mensagem seguinte aguarda a preparação pendente. Falhas não apagam a troca já salva; a próxima tentativa reprocessa a memória.

Em server, o contrato é solicitado com JSON Schema. No CLI, esta versão valida o contrato após a geração, pois a gramática do CLI instalado falhou com um dos GGUF testados. Modelos que não seguem o contrato podem exigir outra configuração ou o modo server.

## Segurança de processos no Windows

Cada worker é criado suspenso, incluído em um Job Object privado com KILL_ON_JOB_CLOSE e só então liberado. Fechar o console, finalizar o HAME ou interromper uma tarefa não deixa os workers gerenciados em execução. Processos já abertos pelo usuário, inclusive outro llama-server.exe, não são adicionados ao Job Object nem encerrados por nome. O navegador padrão não é incluído no grupo dos modelos.

## Configuração portátil

config.example.json é deliberadamente vazio nos caminhos. A configuração real é criada pelo configurador e ignorada pelo Git. Caminhos relativos são resolvidos em relação à pasta de config.json, não ao diretório de trabalho do terminal. Arquivos dentro da pasta do app são salvos como caminhos relativos quando possível. Arquivos externos continuam com seus caminhos locais e precisam ser selecionados novamente em outra máquina.

A configuração permite motor, executáveis, dois GGUF, GPUs, pasta de contexto, modo, porta do chat, contexto, limite de saída e camadas GPU. As portas dos workers server são escolhidas dinamicamente; o chat usa loopback e a porta configurada, por padrão 8088.

## Compilar

Requer **Go 1.23+**, **Python 3** e **clang/LLVM** no PATH. Não há dependências Go externas nem pacotes Python de terceiros. O configurador usa Windows PowerShell 5.1, .NET Framework e Windows Forms disponíveis no Windows; não depende do PowerShell 7.

```powershell
.\build.ps1
```

O script gera o recurso Windows do ícone, executa testes e go vet e cria HAME.exe como aplicativo de console, com versão de arquivo 0.9 RC. Não utiliza -H=windowsgui. config.json é copiado do exemplo se ainda não existir. Recompilar o recurso de ícone requer clang; a arte original está em assets/ e não precisa ser gerada novamente.

Comandos úteis:

```powershell
.\HAME.exe -version
.\HAME.exe -configure
.\HAME.exe -check
.\HAME.exe -probe 'C:\llama\llama-server.exe' -backend server
.\HAME.exe -probe 'C:\llama\llama-cli.exe' -backend cli
.\HAME.exe -open=false
```

-check verifica arquivos e capacidades sem carregar os modelos. O probe consulta o executável escolhido; portanto execute apenas distribuições confiáveis do llama.cpp.

## Código

| Arquivo | Responsabilidade |
| --- | --- |
| main.go | Contratos, compilador, estado e entrada do programa |
| config.go | Configuração portátil, probe e validação de capacidades |
| adapter.go | Adaptadores server/CLI e identidade real do modelo |
| cycle.go | Histórico, API, reset e preparação automática |
| process_windows.go | Job Object, console e ciclo de vida dos workers |
| Configurar-HAME.ps1 | Configurador WinForms com detecção assíncrona |
| *_test.go | Ciclo, compatibilidade, parsing, isolamento de processos |
| webui/ | Interface oficial vendorizada e adaptador de sessões |
| build_icon_resources.py | Ícone e VERSIONINFO nativos do Windows |

## Testes e limites

Veja [docs/VALIDATION.md](docs/VALIDATION.md) para os ensaios reais desta RC e [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md) para a estratégia de atualização.

A memória compilada é textual e com perda; ela não é um KV cache neural. O arquivo original é preservado, mas o resumo pode omitir detalhes. O compilador percorre o histórico em lotes, com custo crescente. Não há promessa de contexto infinito, precisão perfeita ou latência fixa.

O gateway suporta conversas textuais. O SSE atual entrega a resposta completa ao terminar, sem streaming token a token. Imagens, áudio, tools, edição de ramos, regeneração e retomada de streams da interface upstream não estão implementados no gateway. Use uma instância por pasta de dados. A interface vendorizada permanece na versão b10687; atualizar os motores não troca automaticamente essa interface.

## GitHub e licença

Esta pasta é a raiz pronta para o repositório: sem modelos, executáveis, históricos de conversa, logs ou configuração pessoal. [GITHUB.md](GITHUB.md) contém descrição e tópicos sugeridos. Nenhum repositório remoto foi criado ou publicado.

Código próprio: **BSD-3-Clause**, copyright (c) 2026, Busta Supastar. A licença MIT de componentes derivados do llama.cpp está em [LICENSE-llama.cpp.txt](LICENSE-llama.cpp.txt). Consulte [LICENSING.md](LICENSING.md).

O histórico visual pertence ao perfil e à origem (endereço/porta) do navegador. Trocar de perfil ou porta não importa automaticamente esse IndexedDB; os arquivos originais de contexto no disco continuam independentes.
