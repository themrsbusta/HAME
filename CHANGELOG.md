# AlteraÃ§Ãµes

## 0.92 RC â€” 2026-10-27

- Corrigido gatilho: saturaÃ§Ã£o ou recuperaÃ§Ã£o explÃ­cita, sem compactar a cada resposta.
- Contagem real do prompt formatado no server; tokenizer auxiliar e reserva explÃ­cita no CLI.
- Constrained mantÃ©m principal entre interaÃ§Ãµes e alterna apenas ao compactar.
- Resident mantÃ©m ambos os processos; limpa e verifica slot sem descarregar principal.
- context_state volta a ser produzido semanticamente pelo modelo principal.
- CompilaÃ§Ã£o incremental com fontes novas/alteradas e recuperaÃ§Ã£o completa opcional.
- GUI organizada em abas, com polÃ­ticas, reservas, limite de memÃ³ria, prefill, aÃ§Ãµes manuais e teste real de modo.
- Indicadores e controles adicionados como adaptaÃ§Ã£o Ã  WebUI oficial.
- Logotipo do raio substituÃ­do pelo Ã­cone do HAME nos componentes nativos; favicon mantido.
- MigraÃ§Ã£o de configuraÃ§Ãµes e preservaÃ§Ã£o de preferÃªncias; dados antigos nÃ£o modificados.
- Nova pasta independente para rollback da 0.9 RC.

A versÃ£o arquitetural da especificaÃ§Ã£o original continua 1.3; a versÃ£o do aplicativo Ã© 0.92 RC.

### Fotos e memÃ³ria YAML

- Imagens no server com mmproj: original preservado, descriÃ§Ã£o YAML e rota local de recuperaÃ§Ã£o.
- Compilador lÃª YAML; referÃªncias selecionadas preservadas fora da sÃ­ntese, dentro do orÃ§amento.
- Teste real apÃ³s compactaÃ§Ã£o e reinÃ­cio recupera FOTO-091 e link do original.

## Portabilidade 0.92 RC

- Windows x64 preservado com WinForms e detecÃ§Ã£o.
- Linux x64/ARM64 e Termux Android ARM64: configuraÃ§Ã£o manual, sem sondagem na inicializaÃ§Ã£o.
- Grupos de processos Unix e tratamento SIGTERM/SIGHUP/Ctrl+C.
- Exemplos JSON explÃ­citos e scripts de compilaÃ§Ã£o/execuÃ§Ã£o por alvo.
