# Atualização para HAME 1.1 beta 2

Esta atualização remove o limite absoluto de duração das requisições de inferência e de compactação manual. Ela não exige atualização do llama.cpp, troca dos GGUFs nem alteração dos limites de contexto ou geração.

## Instalações existentes

1. Encerre o HAME pelo fluxo normal e confirme que a janela/serviço terminou.
2. Faça uma cópia de segurança de `HAME.exe`.
3. Substitua apenas `HAME.exe` pelo executável beta 2.
4. Mantenha `config.json`, os diretórios de dados, os modelos GGUF, o mmproj e os executáveis do llama.cpp.
5. Inicie o HAME e confira `/health`; a versão deve aparecer como `1.1-beta.2`.

O formato de configuração e os dados de conversa continuam compatíveis com o beta 1. Se preferir rollback, encerre o HAME e restaure o executável guardado. Builds Linux/ARM64 e Termux precisam ser recompilados a partir deste código para receber a correção; nenhum computador remoto é alterado por este pacote.

## Escopo dos limites

O timeout removido aplica-se às inferências, prefill/processamento de prompts e compactação manual. O contexto da requisição ainda permite cancelamento explícito ou por desconexão, e o encerramento do HAME continua encerrando os workers que ele iniciou. Timeout de carga inicial do modelo, sondagem do runtime e execução de ferramentas são controles independentes e foram mantidos.
