// Contract smoke test for the router-core frontend.
//
// This test reads the live OpenRouter trace fixture captured
// against the real lab unit and asserts that the per-turn
// shape matches what the frontend (AiAssistantView, Dashboard)
// expects to consume. It runs with the built-in `node --test`
// runner, so there is no extra dependency to install.
//
// Run it from the repo root:
//
//   node --test frontend/tests/contract.test.mjs
//
// The frontend dev can wire this into their own CI alongside
// vitest; we do not add vitest here because the goal of this
// file is a zero-deps contract check, not a full React render
// test suite.

import { test } from "node:test";
import { readFileSync, readFileSync as _r } from "node:fs";
import { strict as assert } from "node:assert";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(__dirname, "..", "..");
const TRACE = resolve(REPO_ROOT, "fixtures", "agent-traces", "2026-09-04-wifi-exposed.live.jsonl");

function loadTrace() {
  const text = readFileSync(TRACE, "utf8");
  const lines = text.split("\n").filter((l) => l.trim() !== "");
  return lines.map((l) => JSON.parse(l));
}

test("live trace exists and is non-empty", () => {
  const turns = loadTrace();
  assert.ok(turns.length >= 4, `expected >=4 turns, got ${turns.length}`);
});

test("turn 1 is the user question", () => {
  const t = loadTrace()[0];
  assert.equal(t.role, "user");
  assert.equal(t.content, "Is my Wi-Fi exposed?");
});

test("turn 2 is the assistant tool call for wireless", () => {
  const t = loadTrace()[1];
  assert.equal(t.role, "assistant");
  assert.ok(Array.isArray(t.tool_calls), "tool_calls must be an array");
  assert.equal(t.tool_calls.length, 1);
  const tc = t.tool_calls[0];
  assert.equal(tc.function.name, "get_security");
  // The frontend posts {question: "..."} to /v0/chat and the agent
  // dispatches the tool with {name: "..."}. Assert the agent
  // chose the right capability.
  const args = JSON.parse(tc.function.arguments);
  assert.equal(args.name, "wireless");
});

test("turn 3 is the tool result with the parsed wireless body", () => {
  const t = loadTrace()[2];
  assert.equal(t.role, "tool");
  assert.equal(t.name, "get_security");
  // The tool content is a JSON string. The router-core serve
  // wraps the parsed WirelessSecurity struct in {result: ...}.
  const body = JSON.parse(t.content);
  assert.equal(body.result.SSID, "TP-LINK_CBEC16");
  assert.equal(body.result.SecurityType, "wpa2-psk");
  assert.equal(body.result.Cipher, "332");
  assert.ok(body.result.HasPreSharedKey, "HasPreSharedKey must be true");
});

test("turn 4 is the final assistant answer, in Spanish, with a recommendation", () => {
  const t = loadTrace()[3];
  assert.equal(t.role, "assistant");
  assert.ok(t.content.length > 100, "answer must be substantive");
  // Spot-check Spanish content + a recommendation.
  assert.ok(/recomendaci[oó]n/i.test(t.content), "answer must include a recommendation");
  assert.ok(/WPA2|wpa2/i.test(t.content), "answer must reference WPA2");
});
