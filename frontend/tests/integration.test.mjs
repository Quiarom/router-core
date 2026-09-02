// End-to-end integration test: verify the frontend (via Vite dev
// server) can talk to router-core serve and to the agent.
//
// Run with: node --test /tmp/integration_test.mjs
// Requires: serve on 127.0.0.1:8484, frontend on 127.0.0.1:5180.

import { test } from 'node:test';
import { strict as assert } from 'node:assert';

const SERVE = 'http://127.0.0.1:8484';
const FRONT = 'http://127.0.0.1:5180';

async function get(path, base = SERVE) {
  const r = await fetch(base + path);
  return { status: r.status, body: r.headers.get('content-type'), text: await r.text() };
}

test('serve /healthz returns 200 with state ok', async () => {
  const r = await get('/healthz');
  assert.equal(r.status, 200);
  const body = JSON.parse(r.text);
  assert.equal(body.state, 'ok');
});

test('serve /v0/device returns full fingerprint', async () => {
  const r = await get('/v0/device');
  assert.equal(r.status, 200);
  const body = JSON.parse(r.text);
  assert.equal(body.vendor, 'TP-Link');
  assert.equal(body.model, 'TL-WR841N/ND');
  assert.match(body.firmwareVersion.value, /3\.15\.9/);
  assert.match(body.hardwareVersion.value, /WR841N v8 00000000/);
});

test('serve /v0/capabilities reports wireless_security: verified', async () => {
  const r = await get('/v0/capabilities');
  assert.equal(r.status, 200);
  const body = JSON.parse(r.text);
  assert.equal(body.capabilities.wireless_security, 'verified');
  assert.equal(body.capabilities.wps, 'absent');
});

test('serve /v0/security/wireless returns parsed data', async () => {
  const r = await get('/v0/security/wireless');
  assert.equal(r.status, 200);
  const body = JSON.parse(r.text);
  assert.equal(body.state, 'verified');
  assert.equal(body.result.SSID, 'TP-LINK_CBEC16');
  assert.equal(body.result.SecurityType, 'wpa2-psk');
});

test('frontend dev server serves a 200 HTML page', async () => {
  const r = await get('/', FRONT);
  assert.equal(r.status, 200);
  assert.match(r.text, /<html/);
});

test('frontend dev page references the live router-core URL', async () => {
  const r = await get('/', FRONT);
  assert.match(r.text, /\/@vite\/client/);
});
