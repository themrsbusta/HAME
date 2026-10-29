# Arquitetura — HAME 1.1 beta

HAME é um gateway Go entre clientes compatíveis com chat completions e processos externos do llama.cpp. No beta, o backend server obtém a WebUI do llama-server realmente instalado, mantém cache dos assets e injeta `hame-shim.js` antes dos módulos do frontend. O cache é atendido sem adquirir o lock do motor, preservando a interface durante a troca de workers. A WebUI vendorizada permanece embutida como alternativa e para o backend CLI.

O navegador mantém o histórico visual; o script envia somente a nova mensagem user, com identidade estável por conversa. A janela real da API permanece correta. Só a apresentação da WebUI deixa de limitar a conversa pela soma do histórico inteiro. Antes da janela real encher, o gateway sintetiza a memória e reconstrói o KV. Após compactação, uma busca lexical limitada à mesma conversa pode reinjetar trechos originais; eles são transitórios, não substituem os originais nem se acumulam no snapshot ativo.

O principal responde ao usuário e produz o estado semântico da conversa. O compilador sintetiza as fontes persistidas quando o orçamento de contexto exige compactação ou quando há solicitação explícita de recuperação. Originais, histórico e metadados ficam no diretório de dados configurado; sínteses podem omitir informação.

No modo Constrained, o principal permanece ativo entre turnos e os modelos alternam durante a compactação. No modo Resident, os pesos permanecem carregados e o gateway exige mecanismos de slots/KV compatíveis e verificáveis. Configuração, memória, mídia, relógio, API e controle de processos estão separados nos arquivos de `cmd/hame/`.

Windows mantém configurador e sondagem explícita das capacidades do motor. Linux e Android/Termux usam configuração manual, com caminhos, dispositivos, camadas GPU e capacidades declarados. Os motores e modelos não são distribuídos neste repositório.

A especificação original em `specs/HAME_1.3.yaml` descreve a arquitetura pretendida; sua numeração não é a versão do aplicativo e não comprova implementação de todos os objetivos. Ferramentas nativas agora têm um ciclo opcional no chat do principal: descoberta real via `/tools`, validação, execução estruturada, resultado `role=tool` e nova inferência, com limite e registro durável. Compilador, extração de estado e descrição de imagens não executam ferramentas. Referências de fotos não causam nova análise visual e memória sintetizada pode perder detalhes. Consulte `API.md`, `PORTABILITY.md` e `VALIDATION.md`.
