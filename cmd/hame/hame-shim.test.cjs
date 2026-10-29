const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');
const {webcrypto} = require('node:crypto');

function fixture() {
  const calls = [];
  const props = {default_generation_settings: {n_ctx: 4096}, hame: {continuous_history: true}, model_alias: 'Jewel-Q4_K_M.gguf'};
  const window = {fetch: async (input, init) => {
    calls.push({input, init});
    if (new URL(input instanceof Request ? input.url : input, 'http://127.0.0.1:8088/').pathname === '/props') return Response.json(props);
    return Response.json({ok: true});
  }};
  const location = {origin: 'http://127.0.0.1:8088', pathname: '/chat/thread-one'};
  const context = {window, location, document: {baseURI: location.origin + '/', readyState: 'loading', addEventListener() {}}, crypto: webcrypto, URL, Request, Response, Headers, TextEncoder, console};
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, 'hame-shim.js'), 'utf8'), context);
  return {window, location, calls, props};
}

test('UI has continuous history while real server capacity stays truthful', async () => {
  const f = fixture();
  const result = await (await f.window.fetch('/props')).json();
  assert.equal(result.default_generation_settings.n_ctx, null);
  assert.equal(result.hame.real_context_tokens, 4096);
  assert.equal(f.props.default_generation_settings.n_ctx, 4096);
  assert.equal(result.model_alias, 'Jewel-Q4_K_M.gguf');
});

test('native SPA relative API URLs are routed to the HAME gateway', async () => {
  const f = fixture();
  await f.window.fetch('http://127.0.0.1:8088/chat/props?autoload=false');
  assert.equal(new URL(f.calls[0].input).pathname, '/props');
  await f.window.fetch('http://127.0.0.1:8088/chat/thread-one/v1/chat/completions', {body: JSON.stringify({messages: [{role: 'user', content: 'hello'}]})});
  assert.equal(new URL(f.calls[1].input).pathname, '/v1/chat/completions');
});

test('session identity survives subsequent turns and isolates different chats', async () => {
  const f = fixture();
  const body = JSON.stringify({messages: [{role: 'user', content: 'remember Aurora'}], stream: true});
  for (let i = 0; i < 2; i++) await f.window.fetch('/v1/chat/completions', {method: 'POST', body});
  const first = JSON.parse(f.calls[0].init.body), second = JSON.parse(f.calls[1].init.body);
  assert.match(first.session_id, /^ui_[a-f0-9]{64}$/);
  assert.equal(first.session_id, second.session_id);
  f.location.pathname = '/chat/thread-two';
  await f.window.fetch('/v1/chat/completions', {method: 'POST', body});
  assert.notEqual(JSON.parse(f.calls[2].init.body).session_id, first.session_id);
});

test('Request objects and explicit API sessions keep their body and identity', async () => {
  const f = fixture();
  const request = new Request('http://127.0.0.1:8088/v1/chat/completions', {method: 'POST', body: JSON.stringify({session_id: 'explicit-api-session', messages: [{role: 'user', content: 'original'}]})});
  await f.window.fetch(request);
  const body = JSON.parse(f.calls[0].init.body);
  assert.equal(body.session_id, 'explicit-api-session');
  assert.equal(body.messages[0].content, 'original');
});

test('hash-routed native conversations are isolated without an explicit header', async () => {
  const f = fixture(); f.location.pathname = '/'; f.location.hash = '#/chat/hash-one';
  const init = {method:'POST',body:JSON.stringify({messages:[{role:'user',content:'hello'}]})};
  await f.window.fetch('/v1/chat/completions',init);
  f.location.hash = '#/chat/hash-two';
  await f.window.fetch('/v1/chat/completions',init);
  assert.notEqual(JSON.parse(f.calls[0].init.body).session_id,JSON.parse(f.calls[1].init.body).session_id);
});

test('external requests are not rewritten', async () => {
  const f = fixture();
  await f.window.fetch('https://example.org/v1/chat/completions', {method: 'POST', body: '{}'});
  assert.equal(f.calls[0].input, 'https://example.org/v1/chat/completions');
  assert.equal(f.calls[0].init.body, '{}');
});

test('a long browser history is retained locally but not resent to the backend', async () => {
  const f = fixture();
  const messages = [{role:'user',content:'old '.repeat(100000)},{role:'assistant',content:'old answer'},{role:'user',content:'new question'}];
  const body = JSON.stringify({messages});
  await f.window.fetch('/v1/chat/completions',{method:'POST',body});
  const sent = JSON.parse(f.calls[0].init.body);
  assert.equal(sent.messages.length,1);
  assert.equal(sent.messages[0].content,'new question');
  assert.equal(messages.length,3);
  assert.ok(body.length>400000);
  assert.ok(f.calls[0].init.body.length<1000);
});
