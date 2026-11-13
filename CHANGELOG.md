# Alterações

## 1.1 beta 3 — 2026-11-13

- Geração do backend server baseada no espaço livre até 75% da capacidade real de contexto anunciada pelo llama.cpp; os segmentos continuam após compactação, sem limite artificial de saída por turno.
- Monitoramento contínuo dos contadores de prompt/decodificação e capacidade da slot por `/slots`; o painel de uso acompanha a ocupação lógica em tokens.
- Respostas só são concluídas após encerramento natural e ciclo de ferramentas válido quando necessário. Respostas truncadas preservam a tarefa e o texto produzido, compactam e tentam continuar; chamadas de ferramenta incompletas nunca são executadas.
- Falha de capacidade pode acionar uma compactação e uma nova tentativa; geração sem progresso por cinco minutos é cancelada e reportada como travada.
- Configurador Windows agora usa 75% por padrão, grava a política completa por backend e explica que a geração automática depende do llama-server. CLI permanece em limite fixo por chamada.
- Atualizados exemplos de configuração, a especificação YAML da política e a documentação de limites.

## 1.1 beta 2 — 2026-11-12

- Removido o timeout absoluto do cliente HTTP usado em inferências, prefill e processamento prolongado de prompts no llama.cpp.
- Compactação manual também aguarda sem limite de duração; cancelar a solicitação ou encerrar o HAME ainda interrompe o trabalho pelo contexto HTTP/processo.
- Erros HTTP e encerramento inesperado do backend continuam sendo reportados; limites de tokens, contexto, inicialização de modelos e ferramentas permanecem iguais.
- A checagem de saúde do backend durante a carga do modelo passa a ocorrer uma vez por segundo; o encerramento do processo continua acompanhado diretamente.
- Adicionados testes para resposta lenta antes do primeiro token, cancelamento manual e erro do backend.

## 1.1 beta 1 — 2026-10-29

- WebUI obtida do llama-server instalado, com script do HAME e cache persistente de assets; interface embutida anterior como alternativa.
- Histórico contínuo na apresentação, com janela real preservada na API e no painel.
- Compactação automática antes do limite e recuperação lexical de trechos originais por conversa, com orçamento e auditoria.
- Normalização de `--tools all` do configurador, descoberta `/tools` no formato nativo e identidade baseada no GGUF selecionado.
- Testes de cache sem worker, rotas SPA/hash, recuperação após omissão no resumo, isolamento de conversas e limites reais.

## 1.02 — 2026-10-29

- Corrigidos acentos e separadores corrompidos na WebUI e neste histórico.
- Scripts PowerShell com UTF-8 BOM para leitura correta pelo Windows PowerShell 5.1.
- Charset UTF-8 declarado antes dos scripts HTML; testes protegem os textos da interface e a codificação dos scripts.

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
