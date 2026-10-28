# Alterações

## 0.92 RC — 2026-10-27

- Corrigido gatilho: saturação ou recuperação explícita, sem compactar a cada resposta.
- Contagem real do prompt formatado no server; tokenizer auxiliar e reserva explícita no CLI.
- Constrained mantém principal entre interações e alterna apenas ao compactar.
- Resident mantém ambos os processos; limpa e verifica slot sem descarregar principal.
- context_state volta a ser produzido semanticamente pelo modelo principal.
- Compilação incremental com fontes novas/alteradas e recuperação completa opcional.
- GUI organizada em abas, com políticas, reservas, limite de memória, prefill, ações manuais e teste real de modo.
- Indicadores e controles adicionados como adaptação à WebUI oficial.
- Logotipo do raio substituído pelo ícone do HAME nos componentes nativos; favicon mantido.
- Migração de configurações e preservação de preferências; dados antigos não modificados.
- Nova pasta independente para rollback da 0.9 RC.

A versão arquitetural da especificação original continua 1.3; a versão do aplicativo é 0.92 RC.

### Fotos e memória YAML

- Imagens no server com mmproj: original preservado, descrição YAML e rota local de recuperação.
- Compilador lê YAML; referências selecionadas preservadas fora da síntese, dentro do orçamento.
- Teste real após compactação e reinício recupera FOTO-091 e link do original.

## Portabilidade 0.92 RC

- Windows x64 preservado com WinForms e detecção.
- Linux x64/ARM64 e Termux Android ARM64: configuração manual, sem sondagem na inicialização.
- Grupos de processos Unix e tratamento SIGTERM/SIGHUP/Ctrl+C.
- Exemplos JSON explícitos e scripts de compilação/execução por alvo.
