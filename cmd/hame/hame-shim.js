/* HAME 1.1 beta: native llama.cpp UI, continuous history, bounded real KV. */
(() => {
  'use strict';
  const originalFetch = window.fetch.bind(window);
  const currentRoute = () => location.hash?.startsWith('#/') ? location.hash.slice(1) : location.pathname;
  let session = '', anonymous = crypto.randomUUID(), previousPath = currentRoute();
  let runtime = null;
  const resetAnonymous = () => {
    if (currentRoute() !== previousPath) {
      if (!currentRoute().startsWith('/chat/')) anonymous = crypto.randomUUID();
      previousPath = currentRoute();
    }
  };
  const digest = async value => 'ui_' + Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(value))), b => b.toString(16).padStart(2, '0')).join('');
  window.fetch = async (input, init) => {
    const url = new URL(typeof input === 'string' || input instanceof URL ? input : input.url, document.baseURI);
    if (url.origin !== location.origin) return originalFetch(input, init);
    // Some upstream services resolve ./props against the current SPA route.
    const api = url.pathname.match(/\/(props|tools|slots|health|v1\/.*)$/);
    if (api && !url.pathname.startsWith('/hame/')) {
      const canonical = '/' + api[1];
      if (canonical !== url.pathname) {
        url.pathname = canonical;
        input = input instanceof Request ? new Request(url, input) : url.toString();
      }
    }
    if (url.pathname === '/props') {
      const response = await originalFetch(input, init);
      if (!response.ok) return response;
      const props = await response.clone().json();
      if (props.hame?.continuous_history && props.default_generation_settings) {
        // Unknown UI context capacity disables client-side full-history clipping.
        // The actual model window stays truthful in /props and hame.real_context_tokens.
        props.hame.real_context_tokens = props.default_generation_settings.n_ctx;
        props.default_generation_settings.n_ctx = null;
        const headers = new Headers(response.headers);
        headers.delete('Content-Length'); headers.delete('Content-Encoding');
        return new Response(JSON.stringify(props), {status: response.status, headers});
      }
      return response;
    }
    if (url.pathname === '/v1/chat/completions') {
      const source = init?.body ?? (input instanceof Request ? await input.clone().text() : null);
      let body;
      try { body = typeof source === 'string' ? JSON.parse(source) : null; } catch {}
      if (body?.messages?.length) {
        resetAnonymous();
        const headers = new Headers(init?.headers ?? (input instanceof Request ? input.headers : undefined));
        const route = currentRoute().match(/\/chat\/([^/]+)/);
        const identity = body.session_id || headers.get('X-Conversation-Id') || body.conversation_id || route?.[1] || anonymous;
        session = body.session_id || await digest(identity);
        body.session_id = session;
        // The browser retains the complete conversation. Transport only the new
        // user message: HAME already owns its originals, memory and tool trace.
        const last = body.messages[body.messages.length - 1];
        if (last.role === 'user') body.messages = [last];
        headers.set('Content-Type', 'application/json');
        headers.set('X-Conversation-Id', identity);
        // HAME owns the active context and retains the complete original archive.
        return originalFetch(input, {...init, method: 'POST', headers, body: JSON.stringify(body)});
      }
    }
    return originalFetch(input, init);
  };
  async function mount() {
    const panel = document.createElement('div');
    panel.id = 'hame-continuous-history';
    panel.style.cssText = 'position:fixed;top:8px;right:12px;z-index:2147483000;background:#15332d;color:#d4fff0;padding:7px 10px;border-radius:8px;font:12px system-ui;max-width:min(700px,calc(100vw - 32px));box-shadow:0 2px 8px #0003';
    const label = document.createElement('span');
    label.textContent = 'HAME 1.1 beta · conectando';
    const compact = document.createElement('button'); compact.textContent = 'Compactar';
    const recover = document.createElement('button'); recover.textContent = 'Recuperar';
    for (const button of [compact, recover]) button.style.cssText = 'margin-left:10px;cursor:pointer';
    panel.append(label, compact, recover); document.body.append(panel);
    async function refresh() {
      try {
        const response = await originalFetch('/hame/runtime', {cache: 'no-store'});
        if (!response.ok) throw new Error('HAME indisponível');
        runtime = await response.json();
        const cfg = runtime.configuration || {};
        panel.title = 'Configuração: ' + (cfg.config_path || '') + '\nModelo: ' + (cfg.model_path || '');
        let state = runtime.status || {};
        if (session) { const status = await originalFetch('/hame/status?session_id=' + encodeURIComponent(session)); if (status.ok) state = await status.json(); }
        const usage = state.context_usage || {};
        const stages = {compiling: 'compactando memória', extracting_state: 'preparando memória', responding: 'respondendo', loading_context: 'reconstruindo contexto', invalidating_context: 'renovando KV', ready: 'pronto'};
        label.textContent = `HAME ${runtime.version} · ${cfg.model_file || 'modelo não identificado'} · ${cfg.continuous_history ? 'histórico contínuo' : 'contexto limitado'} · KV ${usage.prompt_tokens || 0}/${usage.context_tokens || cfg.real_context_tokens || '?'} · ${stages[state.stage] || state.stage || 'pronto'}`;
        if (cfg.restart_required) label.textContent += ' · configuração alterada: reinicie o HAME';
        if (state.error) label.textContent += ' · ' + state.error;
      } catch (error) { label.textContent = 'HAME · ' + error.message; }
    }
    async function manage(full) {
      if (!session) { label.textContent = 'Envie uma mensagem nesta conversa primeiro.'; return; }
      compact.disabled = recover.disabled = true;
      try {
        const response = await originalFetch('/hame/compact', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({session_id: session, recover: full})});
        if (!response.ok) throw new Error(await response.text());
      } catch (error) { label.textContent = error.message; }
      finally { compact.disabled = recover.disabled = false; }
      await refresh();
    }
    compact.onclick = () => manage(false); recover.onclick = () => manage(true);
    await refresh(); setInterval(refresh, 2000);
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount, {once: true}); else mount();
})();
