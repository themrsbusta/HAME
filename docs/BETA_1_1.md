# HAME 1.1 beta 1

Este documento descreve o beta experimental do HAME 1.1.

O contrato principal envolve a entrega de uma Interface Web (WebUI) nativa do llama-server, complementada por um script adicional. Um recurso chave é a manutenção do cache de assets, o que garante que a interface permaneça estável durante a troca de modelos.

O HAME preserva o histórico original e compacta sua representação textual antes de atingir a janela real de contexto. Ao consultar fatos antigos, busca trechos da própria conversa e os injeta com orçamento limitado. Histórico contínuo não implica entrada infinita nem recuperação perfeita. As ferramentas nativas são executadas pelo HAME; o nome exibido vem do GGUF selecionado na configuração da máquina.

Configuração do beta: `webui: "upstream"`, `continuous_history: true` e `retrieval_tokens: 512`. O modo contínuo compacta automaticamente, inclusive com a política antiga Sob demanda. `continuous_history: false` restaura o comportamento dessa política. `webui: "embedded"` mantém a interface anterior como alternativa; CLI usa essa opção.

O script remove somente o limite do histórico completo na apresentação da WebUI. A API `/props` informa a janela real; o painel HAME mostra sua ocupação. Uma mensagem individual grande demais continua sendo rejeitada antes da inferência. O cache só atende arquivos estáticos e não expõe APIs administrativas do worker.

Limites: busca lexical, sem garantia de recuperação semântica; lê registros originais de até 8 MiB e injeta até o orçamento configurado. Ferramentas de navegador/MCP personalizadas não são executadas pelo gateway; esta integração cobre as ferramentas nativas do servidor. Para editar retroativamente ou ramificar uma conversa, use uma nova conversa neste beta. Linux ARM64/Android precisam de validação em hardware de destino.
