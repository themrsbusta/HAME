# API — HAME 1.1 beta

## Histórico contínuo e WebUI nativa

No beta, `webui: "upstream"` é o padrão do backend server. `GET /` obtém o HTML da WebUI do worker local e injeta `/hame/shim.js`; assets ficam em `data_dir/webui-cache/`, identificados pelo executável/versão do llama e versão do HAME. O cache só atende GET/HEAD de arquivos estáticos, não APIs nem executores do worker. Requisições de chat e metadados permanecem no HAME. CLI usa `webui: "embedded"`; a mesma opção permite voltar à interface anterior no server.

`continuous_history: true` ativa compactação automática antes do limite real, independentemente da política antiga Sob demanda. Após compactação, perguntas seguintes procuram trechos originais da mesma conversa por busca lexical; a injeção é limitada por `retrieval_tokens` (padrão 512, entre 64 e 4096) e pelo espaço disponível. Registros com mais de 8 MiB não entram nessa busca. Evidências ficam em `data_dir/retrieval-audit/`. `continuous_history: false` restaura a aplicação da política de memória anterior.

`/props` continua informando `default_generation_settings.n_ctx` real. Somente a resposta vista pelo JavaScript da WebUI recebe `n_ctx: null` para impedir que o frontend trate o histórico completo como contexto ativo. A janela real continua em `hame.real_context_tokens` e no painel. O backend rejeita uma mensagem individual que não caiba, preservando o histórico. Não há janela real infinita.

`/props.hame` e `/hame/runtime.configuration` identificam o GGUF selecionado, caminho da configuração, ferramentas, diretório inicial, janela real, modo da WebUI e `restart_required`. O nome do arquivo escolhido na máquina prevalece sobre aliases internos. Salvar a configuração não altera um processo já em execução: reinicie o HAME para aplicá-la. Uma segunda instância não carrega modelos antes de detectar que a porta já está ocupada.

`--tools all` ou `--tools=read_file,exec_shell_command` em `main_extra_args` também habilitam o ciclo do HAME. O argumento é normalizado sem duplicação; na ausência de `tool_cwd`, esse modo usa a pasta da configuração como diretório inicial. Se `tools` também estiver presente, as duas seleções precisam coincidir. `GET /tools` retorna a lista nativa de descritores (`tool`, `definition`, `uses_cwd`, permissões), compatível com a WebUI; `POST /tools` continua recusado. As ferramentas são executadas pelo ciclo interno do chat. Ferramentas personalizadas do navegador/MCP não são executadas pelo gateway neste beta.

POST /v1/chat/completions preserva o formato anterior e acrescenta compile e retrieve opcionais. session_id ou X-Conversation-Id identifica a sessão. Apenas a última mensagem user vinda da WebUI é usada; o contexto ativo pertence ao HAME.

```json
{"session_id":"exemplo","messages":[{"role":"user","content":"Qual era o código?"}],"retrieve":true}
```

compile=true solicita compactação manual antes da resposta. retrieve=true solicita recuperação completa nas políticas Sob demanda/Híbrida. O prefixo /recuperar na mensagem equivale a retrieve=true.

POST /hame/compact: {"session_id":"exemplo","recover":false}. Em branco, usa a sessão ativa. recover=true relê o histórico original. A operação é síncrona e não apaga a conversa.

GET /hame/runtime: versão, modo, política, sessão ativa, telemetria e eventos de operação. GET /hame/status?session_id=... preserva os campos anteriores e inclui mode/context_usage. A contagem informa prompt_tokens, reserved_response_tokens, template_reserve_tokens, budget_tokens, budget_percent, exact_prompt e method. exact_prompt=false identifica a reserva de template do CLI ou fallback sem tokenizer.

GET /hame/history preserva as mensagens originais. /health não garante prontidão dos modelos. /props e /v1/models continuam com identidade real do principal. SSE ainda entrega texto completo ao final.

Clientes locais: HAME.exe -memory-status; HAME.exe -compact -session exemplo; HAME.exe -compact -recover. Eles conferem a versão do serviço para não operar silenciosamente em outra instalação.

Falhas de geração/compactação retornam erro HTTP e preservam os arquivos originais. A falta de espaço em Sob demanda exige ação manual antes de inferir.


## Imagens e YAML

Server com `main_mmproj`: a última mensagem user pode conter `content: [{"type":"text","text":"Guarde esta foto"},{"type":"image_url","image_url":{"url":"data:image/png;base64,..."}}]`. O principal gera uma descrição breve antes da conversa textual; foto e YAML ficam em `context/<sessao>/media`. Quatro imagens por mensagem, 10 MiB/40 MP por imagem; corpo JSON total de até 32 MiB. PNG/JPEG/GIF; URLs externas e outros tipos são rejeitados. CLI não oferece este fluxo.

`GET /hame/media?session_id=<sessao>&file=<sha256>.png` recupera os bytes originais; `.yaml` recupera os metadados. Somente nomes de hash e sessões válidas são aceitos. A rota é local, assim como a API do chat; não oferece autenticação entre conversas.

O compilador inclui `.yaml`/`.yml` em suas fontes incrementais. O catálogo fica fora do KV cache e da síntese. Até duas referências pertinentes são reinjetadas deterministicamente na memória compactada. Recuperar a descrição/link não equivale a uma nova análise visual do original.

GET /hame/get_datetime: relógio do sistema, ISO 8601/fuso/offset/Unix; não usa cache. O principal recebe automaticamente uma leitura atualizada a cada turno. Não é um ciclo genérico de chamadas de ferramentas.

## Ferramentas nativas do llama-server

Opt-in no `config.json` (desativado quando `tools` está ausente):

```json
{
  "tools": ["exec_shell_command", "read_file"],
  "tool_cwd": "C:\\HAME\\workspace",
  "tool_max_calls": 8,
  "tool_timeout_seconds": 30
}
```

`["all"]` habilita as ferramentas conhecidas anunciadas pelo servidor: read_file, file_glob_search, grep_search, exec_shell_command, write_file, edit_file e get_info. Requer backend server local gerenciado pelo HAME e suporte real a `--tools`. O HAME acrescenta esse argumento somente ao principal. `tool_cwd` é obrigatório e absoluto quando se usa apenas o campo JSON `tools`; define o diretório inicial, **não** uma sandbox. Comandos herdam os privilégios do processo; habilite apenas em estação local confiável.

`GET /tools` descobre e lista as definições reais do worker. `POST /tools` no HAME é recusado: o executor nativo não é exposto como proxy HTTP. O chat descobre as definições, envia `tools`, valida IDs, nomes e argumentos estruturados, executa cada chamada e devolve `role=tool`/`tool_call_id` antes de continuar a geração. Texto livre nunca é interpretado como comando. Descrições de imagens, extração de estado e compilador não executam ferramentas. Clientes do chat continuam recebendo a resposta textual final, inclusive em SSE; o histórico contém o ciclo completo.

Limite por turno: 1–32 chamadas (padrão 8). Timeout: 1–120 s (padrão 30); o timeout nativo de shell é limitado por essa configuração. O prazo HTTP das outras ferramentas não garante cancelamento de efeitos no servidor. Falha de transporte não gera repetição automática: uma operação pode ter ocorrido. Cada chamada é registrada antes e depois em `data_dir/tool-audit/<sessao>/*.json`; em falha após execução, consulte esse registro antes de repetir a tarefa. Saída de comando com exit code diferente de zero continua sendo um resultado de ferramenta, não sucesso presumido.

O ciclo inclui as definições no orçamento de contexto e interrompe antes de uma nova inferência se não houver espaço. Resultados já executados permanecem no audit mesmo se o turno falhar. O serviço continua em loopback, recusa Host externo, Origin de outro site, chamadas cross-site e POST sem application/json quando ferramentas estão habilitadas. Não há autenticação contra outros programas locais: quem pode chamar o chat pode solicitar ações com os privilégios do HAME. A API interna `/tools` do llama.cpp é experimental e precisa ser revalidada ao trocar de build; validada localmente com b11946.
