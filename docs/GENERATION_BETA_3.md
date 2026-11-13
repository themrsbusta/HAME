# Geração e recuperação no HAME 1.1 beta 3

## Backend llama-server

O beta 3 mede o prompt já formatado com a API do llama-server e consulta `/slots` para obter a capacidade real da janela (`n_ctx`) e os contadores da geração. Cada trecho recebe um orçamento calculado até 75% da capacidade. A fronteira de tokens limita o trecho atual, não o tamanho total da resposta: quando alcançada, o HAME registra o trecho, compacta a conversa e solicita continuação. Assim, não existe um teto artificial único de tokens por turno.

A compactação mantém as instruções de sistema e o pedido ativo, preserva o estado necessário das ferramentas, resume o histórico antigo e reconstrói o contexto do runtime. A conversa visível continua no arquivo; o KV da slot pode ser limpo ou reconstruído pelo mecanismo já configurado para o modo de execução.

O contador de ocupação representa tokens lógicos observados na slot (`n_prompt_tokens + n_decoded`) divididos por `n_ctx`. Ele não mede bytes de KV nem consumo exato de VRAM. O monitor atualiza o estado durante a geração. Se os contadores não avançarem por cinco minutos enquanto a slot continuar processando, o HAME cancela a requisição e sinaliza geração travada. Cancelamentos do cliente também interrompem a requisição.

Uma resposta só é salva como concluída quando o llama-server informa encerramento natural (`finish_reason: stop`) e há texto final. Se houver ferramentas, a resposta intermediária precisa encerrar com `tool_calls`, cada chamada deve passar pela validação e execução autorizada, e o modelo precisa produzir depois uma resposta natural. Um pedido de ferramenta truncado nunca é executado.

Quando o backend rejeita uma geração por falta de capacidade, o HAME tenta uma compactação e uma repetição. Uma resposta interrompida por limite de segmento é preservada como ponto de continuação, compactada e retomada. O texto parcial segue no arquivo de conversa e é enviado de volta como referência para que o modelo continue sem repeti-lo. Se uma continuação repetir exatamente o mesmo trecho, o turno termina como incompleto em vez de entrar em ciclo.

## Backend llama-cli

O CLI não expõe a slot ativa por `/slots` nem permite monitorar ocupação durante a inferência. Por isso, seu perfil usa contexto configurado, saída e reserva fixas, sem compactação automática por ocupação KV e sem detecção de travamento pelo HAME. A resposta é marcada incompleta se atingir o limite de saída configurado. Use `llama-server` para a política automática descrita acima.

## Configuração

Os blocos `generation`, `context`, `completion` e `safety` dos arquivos `configs/*.json` espelham a política em [specs/HAME_1.1-beta.3-generation.yaml](../specs/HAME_1.1-beta.3-generation.yaml). Para o backend server, use:

- `generation.output_tokens: "auto"` e `reserve_tokens: "auto"`;
- `context.capacity: "model_runtime"` e monitoramento KV habilitado;
- compactação por `kv_cache_occupancy` em 75%;
- `artificial_output_limit: false` e `truncated_response.mark_completed: false`.

Os campos numéricos antigos `output_tokens` e `reserve_tokens` permanecem nos JSONs por compatibilidade com o configurador e com outras rotinas de orçamento. No perfil server beta 3, a política aninhada controla a geração do chat principal; as funções auxiliares de compilação continuam limitadas por seus próprios orçamentos.
