# API — HAME 1.0

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
