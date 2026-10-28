# Arquitetura — HAME 1.0

HAME é um gateway Go entre clientes compatíveis com chat completions e processos externos do llama.cpp. A WebUI vendorizada é embutida no binário por `cmd/hame/main.go`, com `//go:embed all:webui`. O gateway serve os arquivos e adapta a conversa ao contexto ativo da sessão.

O principal responde ao usuário e produz o estado semântico da conversa. O compilador sintetiza as fontes persistidas quando o orçamento de contexto exige compactação ou quando há solicitação explícita de recuperação. Originais, histórico e metadados ficam no diretório de dados configurado; sínteses podem omitir informação.

No modo Constrained, o principal permanece ativo entre turnos e os modelos alternam durante a compactação. No modo Resident, os pesos permanecem carregados e o gateway exige mecanismos de slots/KV compatíveis e verificáveis. Configuração, memória, mídia, relógio, API e controle de processos estão separados nos arquivos de `cmd/hame/`.

Windows mantém configurador e sondagem explícita das capacidades do motor. Linux e Android/Termux usam configuração manual, com caminhos, dispositivos, camadas GPU e capacidades declarados. Os motores e modelos não são distribuídos neste repositório.

A especificação original em `specs/HAME_1.3.yaml` descreve a arquitetura pretendida; sua numeração não é a versão do aplicativo e não comprova implementação de todos os objetivos. A 1.0 preserva as limitações da 0.92 RC: não executa ferramentas genéricas, referências de fotos não causam nova análise visual e memória sintetizada pode perder detalhes. Consulte `API.md`, `PORTABILITY.md` e `VALIDATION.md`.
