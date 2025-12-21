<script setup>
import { ref, reactive, computed, onBeforeUnmount } from "vue";
import { search, answer, WS_BASE } from "../api.js";
import StatusStepper from "../components/StatusStepper.vue";

const form = reactive({ query: "", topK: 5 });

const loading = ref(false);     // HTTP/搜索 loading
const statusText = ref("");     // 文字状态
const stage = ref("idle");      // connecting | hf_bge | vector_search | llm | done | error | idle | connected
const hits = ref([]);
const output = ref("");         // 流式/非流式统一写到这里
const error = ref("");
const useStreaming = ref(true); // 有 WS 就开；没有就关

let ws = null;

function resetView() {
  statusText.value = "";
  hits.value = [];
  output.value = "";
  error.value = "";
  stage.value = "idle";
}

async function runSearchOnly() {
  resetView();
  loading.value = true;
  try {
    // 仅检索
    statusText.value = "searching…";
    try {
      const res = await search(form.query, form.topK);
      hits.value = res?.hits || res?.items || [];
      statusText.value = `done: ${hits.value.length} hit(s)`;
    } catch { /* 如果没有 /search 就忽略 */ }

    // （可选）非流式回答：若你的后端有 /answer
    try {
      const resAns = await answer(form.query, form.topK);
      // 兼容不同返回结构
      output.value = typeof resAns === 'string' ? resAns : (resAns?.text || resAns?.answer || JSON.stringify(resAns, null, 2));
      statusText.value = "done";
      stage.value = "done";
    } catch { /* 没有 /answer 就只显示检索 */ }
  } catch (e) {
    error.value = e?.message || String(e);
  } finally {
    loading.value = false;
  }
}

function connectWS() {
  // 你的后端路径是 /ws/answer（保留）
  const url = `${WS_BASE}/ws/answer`;
  try { ws?.close(); } catch {}
  output.value = "";
  stage.value = "connecting";
  statusText.value = "connecting…";

  ws = new WebSocket(url);

  ws.onopen = () => {
    stage.value = "connected";
    statusText.value = "connected";
    // 发送 ask
    ws.send(JSON.stringify({ type: "ask", query: form.query, top_k: form.topK }));
  };

  ws.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data);
      // 你的服务端协议：
      // status: {stage, msg} | hits: {items} | delta: {text} | error | done
      if (msg.type === "status") {
        // 将服务端 stage 透传给 Stepper
        // 期望值：hf_bge | vector_search | llm | done
        if (msg.stage) stage.value = msg.stage;
        statusText.value = `[${msg.stage || 'status'}] ${msg.msg || ''}`.trim();
      }
      if (msg.type === "hits") hits.value = msg.items || [];
      if (msg.type === "delta") output.value += msg.text || "";
      if (msg.type === "error") {
        error.value = msg.msg || "unknown error";
        stage.value = "error";
      }
      if (msg.type === "done") {
        stage.value = "done";
        statusText.value = "done";
        ws?.close();
      }
    } catch {
      // 兼容纯文本 delta
      stage.value = "llm";
      output.value += ev.data;
    }
  };

  ws.onerror = () => { error.value = "ws error"; stage.value = "error"; };
  ws.onclose = () => { /* noop */ };
}

async function run() {
  resetView();
  if (useStreaming.value) {
    connectWS();
  } else {
    await runSearchOnly();
  }
}

function copyAnswer() {
  navigator.clipboard.writeText(output.value || "");
}

onBeforeUnmount(() => { try { ws?.close(); } catch {} });

const hasHits = computed(() => hits.value && hits.value.length > 0);
const isBusy = computed(() =>
  loading.value || ["connecting","connected","hf_bge","vector_search","llm"].includes(stage.value)
);
</script>

<template>
  <div class="page">
    <header class="head">
      <h1>RAG Console</h1>
      <div class="right">
        <StatusStepper :stage="stage" />
        <label class="tog"><input type="checkbox" v-model="useStreaming" /> stream via WS</label>
      </div>
    </header>

    <section class="panel">
      <div class="row">
        <input class="q" v-model="form.query" placeholder="Type your question…" />
        <input class="k" type="number" v-model.number="form.topK" min="1" max="20" />
        <button :disabled="isBusy || !form.query" @click="run">Run</button>
      </div>
      <div class="status" v-if="statusText">{{ statusText }}</div>
      <div class="error" v-if="error">Error: {{ error }}</div>
    </section>

    <section class="grid">
      <div class="card">
        <h2>Top-K Hits</h2>
        <div v-if="!hasHits" class="muted">no hits</div>
        <ol v-else>
          <li v-for="(h, i) in hits" :key="h.id ?? i" class="hit">
            <div class="id">#{{ i+1 }} • {{ h.id ?? h.doc_id ?? '—' }}</div>
            <div class="score" v-if="h.score != null">score: {{ Number(h.score).toFixed(3) }}</div>
            <div class="text" v-if="h.text">{{ h.text }}</div>
            <details v-if="h.metadata"><summary>metadata</summary><pre>{{ h.metadata }}</pre></details>
          </li>
        </ol>
      </div>

      <div class="card">
        <h2>Answer</h2>
        <div class="answer">
          <span v-if="!output && stage==='llm'" class="caret"></span>
          {{ output }}
        </div>
        <div class="actions">
          <button @click="copyAnswer" :disabled="!output">Copy</button>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.page { max-width: 1100px; margin: 24px auto; padding: 0 16px; font-family: ui-sans-serif, system-ui; }
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.right { display:flex; align-items:center; gap:10px; }
.tog { font-size: 14px; color: #444; }
.panel { background: #fafafa; border: 1px solid #eee; padding: 12px; border-radius: 8px; margin-bottom: 16px; }
.row { display: grid; grid-template-columns: 1fr 90px 120px; gap: 8px; }
.q, .k { padding: 10px; border: 1px solid #ddd; border-radius: 6px; }
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
.answer { min-height: 200px; white-space: pre-wrap; position: relative; }
.muted { color: #999; }
.actions { margin-top: 8px; }

/* 流式光标 */
.caret{
  width: 10px; height: 18px; display: inline-block; margin-right: 4px;
  background: #6366f1; animation: blink 1s steps(1) infinite; vertical-align: -3px;
}
@keyframes blink{ 50%{ opacity: 0 } }
</style>
