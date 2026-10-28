# HAME 0.91 RC

<p align="center"><img src="assets/HAME.png" alt="HAME" width="170"></p>

**Hierarchical Async Memory Engine — memória persistente sobre llama.cpp, agora com compactação por ocupação e modos Constrained/Resident distintos.**

Esta revisão é incremental sobre a 0.9 RC. Mantém server/CLI, WebUI oficial vendorizada, histórico persistente, configurador WinForms, console e isolamento dos workers por Job Object. A especificação original de Busta Supastar em specs/HAME_1.3.yaml não foi alterada. Código próprio sob BSD-3-Clause.

## O que foi corrigido

A 0.9 RC compilava a memória e encerrava o principal após cada resposta. A 0.91 RC mantém a conversa ativa enquanto a próxima requisição couber no orçamento. Antes de inferir, conta o prompt, considera a saída reservada e a margem de template e decide se precisa compactar. Não há compilação obrigatória pós-resposta.

O principal produz context_state semântico com session_id, current_goal, unresolved_question, last_summary e search_intent. O orquestrador valida os campos, sem substituir a intenção por um objetivo genérico. O compilador recebe esse estado, a memória consolidada anterior e as fontes persistentes; produz compiled_prompt com system_instruction, context_bundle e response_format, usados na próxima memória ativa.

## Modos de execução

| Modo | Conversa normal | Compactação |
| --- | --- | --- |
| **Constrained** | Principal server permanece carregado | Principal extrai estado e encerra; compilador carrega, compila e encerra; principal reinicia com a memória |
| **Resident** | Principal e compilador server permanecem carregados | Compilador trabalha; a slot 0 do principal é apagada e verificada; memória é substituída sem trocar PID/pesos |
| **CLI por tarefas** | Cada inferência é um processo, como na 0.9 RC | Só compila quando a política pede; nunca anuncia residência ou prefill persistente |

No Constrained os dois modelos não são mantidos carregados ao mesmo tempo. No Resident, não se mata o principal para limpar o KV cache. A versão instalada deve oferecer slots, erase e mecanismos para impedir cache antigo fora da slot. O HAME habilita --slots, --slot-save-path em pasta própria, --cache-ram 0 e --no-cache-idle-slots. No startup, faz um pequeno prefill de teste, exige cache não vazio, chama erase e confirma n_prompt_tokens=0 antes de aceitar Resident. Backend incompatível é rejeitado, não simulado.

O prefill de compatibilidade é um teste curto de suporte; a opção de prefill da memória compilada controla as compactações da conversa.

## Políticas

- **Automática:** compacta ao atingir o percentual configurado. Recuperação em uma pergunta via flag/comando exige Híbrida ou Sob demanda; a ação manual completa continua disponível.
- **Sob demanda:** compacta por solicitação explícita. Se a próxima entrada não couber, bloqueia antes de inferir e pede compactação.
- **Híbrida:** aceita saturação e recuperação explícita.

Ações manuais estão disponíveis no chat e no configurador. Use **Compactar** para consolidar incrementalmente e **Recuperar histórico** para reler as fontes originais. Em uma pergunta, use retrieve=true na API ou o prefixo /recuperar seguido da pergunta para solicitar recuperação completa, conforme a política.

## Contagem e limites

No server, /apply-template formata as mensagens e /tokenize conta os tokens reais, incluindo instruções e template. O orçamento considera prompt + reserva de resposta + margem adicional. A reserva deve cobrir output_tokens. A memória compilada possui limite próprio e seu tamanho é validado com o tokenizer.

No CLI, HAME detecta llama-tokenize.exe ao lado do motor e usa o vocabulário do GGUF. A contagem do conteúdo é real, mas o CLI por tarefas não expõe o prompt formatado antes da geração ao gateway. Por isso o total é marcado como **não exato**, com uma reserva de template configurável. Se o tokenizer auxiliar não estiver disponível, usa um limite superior conservador por bytes UTF-8, identificado no status; não apresenta esse fallback como contagem real. Templates personalizados podem precisar de reserva maior. Para orçamento exato do prompt formatado, use server.

Mensagens que sozinhas excedem o orçamento são recusadas antes de geração. Se memória compilada + nova mensagem não couberem, a requisição falha sem substituir o snapshot ativo. A capacidade real do modelo/backend continua valendo; o tamanho selecionado deve ser compatível com o modelo e hardware.

## Configurador

Execute Configurar-HAME.cmd ou HAME.exe -configure. A seção **Gerenciamento de Memória e Execução** oferece:

| Campo | Padrão |
| --- | --- |
| Janela | 8192 tokens |
| Gatilho | 90% do orçamento |
| Reserva para resposta | Saída máxima configurada |
| Reserva de template | 128 tokens |
| Política | Híbrida |
| Memória compilada máxima | 1024 tokens, ajustada em janelas pequenas |
| Prefill compilado | Ativado para server |
| Modo | Constrained |

Mantém executável, backend, dois GGUF, CPU/GPU independente por modelo, pasta de dados, porta, saída máxima e camadas GPU. A detecção consulta a instalação escolhida. Resident e prefill residente não são habilitados para CLI. **Testar** verifica arquivos e parâmetros sem carregar pesos; **Testar modo selecionado** carrega os modelos e verifica os mecanismos reais, podendo demorar. Evite executar outro teste de modo enquanto sua instalação já ocupa a memória com uma conversa.

**Consultar contexto** mostra a ocupação da conversa ativa. **Compactar agora** atua no HAME já aberto no endereço configurado; marque recuperação completa para reler o arquivo original. Sessão em branco seleciona a última conversa ativa.

Configurações 0.9 sem os novos campos recebem defaults em memória. Caminhos, modelos, dispositivos e preferências não são apagados. Ao salvar pela GUI, campos desconhecidos também são preservados. O carregamento não sobrescreve o arquivo antigo automaticamente.

## Histórico e eficiência

O texto original não é apagado, truncado ou substituído pela compactação. data/context contém trocas originais; data/archive de versões anteriores continua legível. O snapshot de trabalho fica em data/sessions. A WebUI mantém o histórico visual no IndexedDB e não recebe instrução para apagá-lo.

A compilação normal reaproveita o contrato anterior e envia apenas arquivos novos/alterados, detectados por tamanho e modificação. A recuperação completa relê originais, inclusive os já consolidados, para permitir buscar fatos omitidos do resumo. A base global data/knowledge também participa. Referências de origem são preservadas no contrato; o cache de fontes é operacional, não substitui o arquivo original.

Há perda possível na síntese e custo crescente de recuperação. Não se promete contexto literalmente infinito nem precisão perfeita. Uma única instância deve usar cada pasta de dados. O histórico visual depende do perfil e da origem do navegador; o disco é independente.

## Iniciar e compilar

Windows x64, distribuição completa do llama.cpp e dois GGUF. O ZIP não inclui motores, DLLs ou modelos. Vulkan exige GPU/driver compatíveis; CPU é opção independente.

```powershell
.\Configurar-HAME.cmd
.\Iniciar-HAME.cmd
```

O console controla somente os workers que HAME iniciou. Fechá-lo ou usar Ctrl+C encerra esses workers, preservando processos independentes. Não há tray. O nome real do modelo aparece na WebUI e o favicon é do HAME.

Para compilar: Go 1.23+, Python 3 e clang no PATH. O configurador usa Windows PowerShell 5.1/Windows Forms. Não foram adicionadas dependências Go externas.

```powershell
.\build.ps1
.\HAME.exe -check
.\HAME.exe -test-mode -open=false
```

A especificação permanece em specs/. Consulte docs/VALIDATION.md para resultados reais, incluindo perdas ou limitações observadas. Consulte docs/API.md para as ações manuais e telemetria.

## Limitações preservadas

Gateway textual, SSE com resposta completa ao final, sem tools, áudio, edição de ramos ou retomada de streams. A WebUI vendorizada é b10687; trocar o motor não a atualiza. Versões futuras de llama.cpp podem exigir adaptação. O CLI não é residente e sua validação JSON é posterior à geração. O arquivo original e o snapshot são gravações separadas, não uma transação única de banco.

Código próprio BSD-3-Clause, copyright (c) 2026 Busta Supastar. Avisos MIT upstream preservados em LICENSE-llama.cpp.txt. Nenhum repositório remoto foi criado ou publicado automaticamente.

Contratos inválidos podem receber uma única tentativa corretiva, sem preencher objetivos semanticamente por conta do orquestrador. Identificadores alfanuméricos em maiúsculas com hífen/underscore são conferidos literalmente contra a evidência; grafias novas são corrigidas pelo modelo ou rejeitadas. Isso não valida toda a semântica e não elimina a possibilidade de perdas. Saídas inválidas de diagnóstico ficam na pasta privada de sessões, não no pacote público.

## Argumentos extras de inicialização

O configurador possui main_extra_args para o modelo principal, por exemplo --tools all ou --threads 8. Valores com espaços usam aspas duplas. O texto é convertido em argv e passado diretamente ao executável, sem cmd, PowerShell ou avaliação de scripts. O compilador não recebe esses argumentos extras. Opções de modelo, contexto, portas, arquivos e isolamento gerenciadas pelo HAME são recusadas em conflito. Flags longas precisam ser anunciadas pelo executável selecionado.

O campo configura o motor; ele não implementa chamadas de ferramentas no gateway. --tools all pode habilitar recursos internos do llama escolhido, mas o suporte a tools/ciclos tool-call na API HAME continua fora desta RC.

## Argumentos extras (principal)

No configurador, preencha o campo com opções como `--tools all`. Caminhos com espaços devem estar entre aspas. O HAME converte o texto em argumentos sem executar shell e verifica o suporte anunciado pelo executável. Opções de modelo, contexto, dispositivos, porta e cache são geridas pelo HAME. O campo aplica-se somente ao principal. O gateway continua oferecendo conversa textual; habilitar ferramentas no backend não adiciona execução de ferramentas ao HAME.

## Projetor e detecção automática

O campo opcional **Projetor mmproj** (`main_mmproj`) carrega um GGUF de projeção compatível somente no modelo principal. O HAME verifica o arquivo e o suporte a `--mmproj`. No modo server, também habilita o fluxo de fotos com memória YAML descrito abaixo.

O configurador detecta dispositivos ao abrir com um executável configurado e ao alterar seu caminho. Em **Automático**, cada inicialização do HAME consulta o llama selecionado e usa a primeira GPU anunciada pelo backend; sem dispositivos, usa CPU. Se o backend não oferecer `--device`, sua própria seleção é preservada. Escolhas manuais de GPU/CPU continuam respeitadas. A seleção automática não estima se os dois modelos cabem juntos na VRAM; use Testar modo para verificar Resident.

## Fotos com memória YAML

Com llama-server e `main_mmproj` configurado, o chat aceita imagens PNG/JPEG/GIF em mensagens OpenAI `image_url` com data URL base64. A interface anuncia visão quando esse modo está disponível. Até quatro imagens por mensagem, 10 MiB por arquivo e 40 megapixels. Endereços remotos não são baixados.

O principal produz uma descrição breve. O HAME preserva os bytes originais em `data/context/<sessao>/media/<sha256>.<ext>` e cria `<sha256>.yaml` com descrição, hash, modelo/projetor e `image_uri`. O compilador lê YAML junto dos demais arquivos de contexto e preserva referências pertinentes durante a compactação. A foto não é embutida no resumo nem passada ao compilador textual.

`GET /hame/media?session_id=<sessao>&file=<sha256>.png` devolve o original; a mesma rota aceita o arquivo `.yaml`. A recuperação da memória trabalha com descrição e referência. Reanalisar visualmente o original exige enviá-lo novamente ao modelo; a referência sozinha não faz o modelo enxergar a foto. A descrição pode omitir ou inventar detalhes e é marcada como gerada pelo modelo. CLI não aceita anexos neste fluxo.

O contexto normal da conversa e o compilador continuam textuais, usando a descrição e o vínculo ao original.

Para não perder o vínculo por omissão do compilador, o HAME reinjeta até duas referências de fotos no contrato compilado, por correspondência de palavras da intenção de busca com a descrição; desempate pela data mais recente. O catálogo YAML completo permanece no disco. O orçamento de memória é respeitado: primeiro reduz trechos sintetizados, preservando os vínculos; se eles ainda não couberem, a compactação falha sem substituir o snapshot ativo.

## Data e hora do sistema

O HAME fornece ao principal uma leitura de data/hora local do sistema (`HAME_RUNTIME_DATETIME`) em cada mensagem, incluindo após compactação. A leitura substitui a anterior na instrução de sistema e entra no orçamento de tokens. Funciona em server e CLI, sem depender de tool-calling ou --tools all. O modelo recebe o instante da requisição, não um relógio continuamente atualizado.

`GET /hame/get_datetime` consulta a mesma fonte e retorna datetime ISO 8601, date, time, timezone, utc_offset_seconds, unix e source=system_clock. A rota não usa cache. O relógio e o fuso são os da máquina onde o HAME executa. Esta funcionalidade não implementa execução genérica de ferramentas nem registra get_datetime no endpoint /tools.
