<script setup>
import { ref, reactive, computed, onBeforeUnmount } from "vue";
import { search } from "../api";

const WS_BASE = import.meta.env.VITE_WS_BASE;

const form = reactive({ query: "", topK: 5 });
const loading = ref(false);
const statusText = ref("");
const hits = ref([]);
const answer = ref("");
const error = ref("");
const useStreaming = ref(true); // 你后端 WS 就绪后打开；没就先关

let ws = null;

function resetView() {
  statusText.value = "";
  hits.value = [];
  answer.value = "";
  error.value = "";
}

async function runSearchOnly() {
  resetView();
  loading.value = true;
  try {
    statusText.value = "searching…";
    const res = await search(form.query, form.topK);
    hits.value = res?.hits || [];
    statusText.value = `done: ${hits.value.length} hit(s)`;
  } catch (e) {
    error.value = e?.message || String(e);
  } finally {
    loading.value = false;
  }
}

function connectWS() {
  const url = `${WS_BASE}/ws/answer`;
  ws = new WebSocket(url);
  ws.onopen = () => {
    // 发送 ask
    ws.send(JSON.stringify({ type: "ask", query: form.query, top_k: form.topK }));
  };
  ws.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data);
      if (msg.type === "status") statusText.value = `[${msg.stage}] ${msg.msg}`;
      if (msg.type === "hits") hits.value = msg.items || [];
      if (msg.type === "delta") answer.value += msg.text || "";
      if (msg.type === "error") error.value = msg.msg || "unknown error";
      if (msg.type === "done") {
        statusText.value = "done";
        ws?.close();
      }
    } catch (e) {
      // 兼容纯文本 delta
      answer.value += ev.data;
    }
  };
  ws.onerror = () => { error.value = "ws error"; };
  ws.onclose = () => { /* noop */ };
}

async function run() {
  resetView();
  if (useStreaming.value) {
    connectWS();
  } else {
    // 仅检索展示 +（可选）非流式回答
    await runSearchOnly();
  }
}

function copyAnswer() {
  navigator.clipboard.writeText(answer.value || "");
}

onBeforeUnmount(() => {
  try { ws?.close(); } catch {}
});

const hasHits = computed(() => hits.value && hits.value.length > 0);
</script>

<template>
  <div class="page">
    <header class="head">
      <h1>RAG Console</h1>
      <div class="toggles">
        <label><input type="checkbox" v-model="useStreaming" /> stream via WebSocket</label>
      </div>
    </header>

    <section class="panel">
      <div class="row">
        <input class="q" v-model="form.query" placeholder="Type your question…" />
        <input class="k" type="number" v-model.number="form.topK" min="1" max="20" />
        <button :disabled="loading || !form.query" @click="run">Run</button>
      </div>
      <div class="status" v-if="statusText">{{ statusText }}</div>
      <div class="error" v-if="error">Error: {{ error }}</div>
    </section>

    <section class="grid">
      <div class="card">
        <h2>Top-K Hits</h2>
        <div v-if="!hasHits" class="muted">no hits</div>
        <ol v-else>
          <li v-for="(h, i) in hits" :key="h.id" class="hit">
            <div class="id">#{{ i+1 }} • {{ h.id }}</div>
            <div class="score">score: {{ (h.score ?? 0).toFixed(3) }}</div>
            <div class="text" v-if="h.text">{{ h.text }}</div>
            <details v-if="h.metadata">
              <summary>metadata</summary>
              <pre>{{ h.metadata }}</pre>
            </details>
          </li>
        </ol>
      </div>

      <div class="card">
        <h2>Answer (DeepSeek)</h2>
        <div class="answer">{{ answer }}</div>
        <div class="actions">
          <button @click="copyAnswer" :disabled="!answer">Copy</button>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.page { max-width: 1100px; margin: 24px auto; padding: 0 16px; font-family: ui-sans-serif, system-ui; }
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.toggles { font-size: 14px; color: #444; }
.panel { background: #fafafa; border: 1px solid #eee; padding: 12px; border-radius: 8px; margin-bottom: 16px; }
.row { display: grid; grid-template-columns: 1fr 90px 120px; gap: 8px; }
.q { padding: 10px; border: 1px solid #ddd; border-radius: 6px; }
.k { padding: 10px; border: 1px solid #ddd; border-radius: 6px; }
button { padding: 10px 14px; border: 0; border-radius: 6px; background: #111827; color: #fff; cursor: pointer; }
button[disabled] { opacity: .5; cursor: not-allowed; }
.status { margin-top: 8px; font-size: 13px; color: #555; }
.error { margin-top: 8px; color: #b91c1c; font-weight: 600; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.card { border: 1px solid #eee; border-radius: 8px; padding: 12px; background: #fff; min-height: 260px; }
.hit { padding: 8px 0; border-bottom: 1px dashed #eee; }
.hit:last-child { border-bottom: 0; }
.id { font-weight: 600; }
.score { font-size: 12px; color: #666; }
.text { margin-top: 6px; white-space: pre-wrap; }
.answer { min-height: 200px; white-space: pre-wrap; }
.muted { color: #999; }
.actions { margin-top: 8px; }
</style>
