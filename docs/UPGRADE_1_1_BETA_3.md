# Atualização para HAME 1.1 beta 3

O beta 3 adiciona geração segmentada e recuperação automática ao backend `llama-server`. Guarde uma cópia do executável e do `config.json` antes de atualizar. Conversas e modelos GGUF existentes permanecem em suas pastas.

## Backend llama-server

Com `generation` ausente ou configurado para `auto`, o HAME consulta o contexto real da slot, monitora a ocupação lógica em tokens e compacta a 75%. A geração continua em novos segmentos até um encerramento natural. Uma resposta interrompida permanece incompleta e pode ser retomada após compactação.

Os números antigos `output_tokens` e `reserve_tokens` no JSON continuam sendo aceitos. Para o chat principal, `generation.output_tokens: "auto"`, `generation.reserve_tokens: "auto"` e `artificial_output_limit: false` ativam a política nova. O configurador Windows grava essa configuração automaticamente quando `llama-server` está selecionado.

## Backend llama-cli

O CLI mantém valores fixos de geração por chamada. O monitoramento KV e a compactação automática não estão disponíveis nesse backend; selecione `llama-server` para usar a política beta 3 integral.

Confirme `/health` após iniciar e consulte `docs/GENERATION_BETA_3.md` para as condições de conclusão, cancelamento, detecção de travamento e recuperação de falta de recursos.
