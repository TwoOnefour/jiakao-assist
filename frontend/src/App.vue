<script setup>
import { ref, reactive, computed, onMounted, nextTick, onBeforeUnmount } from "vue";
import StatusStepper from "./components/StatusStepper.vue";
import { search, answer, WS_BASE } from "./api";

const sessions = ref([{ id: 's-1', name: '新会话', messages: [] }]);
const currentSid = ref('s-1');
const cur = computed(() => sessions.value.find(s => s.id === currentSid.value));

const form = reactive({ query: "", topK: 5 });
const stage = ref("idle");
const statusText = ref("");
const hits = ref([]);
const streamOut = ref("");
const error = ref("");
const loading = ref(false);
const useStreaming = ref(true);

let ws = null;
const scrollRef = ref(null);

function pushMsg(role, content) {
  cur.value.messages.push({ id: role[0] + '-' + Date.now() + Math.random().toString(36).slice(2,6), role, content });
  autoScroll();
}
async function autoScroll() {
  await nextTick();
  const el = scrollRef.value;
  if (el) el.scrollTo({ top: el.scrollHeight + 999, behavior: 'smooth' });
}
function resetForRun() {
  statusText.value = ""; hits.value = []; streamOut.value = ""; error.value = ""; stage.value = "idle";
}

// REST 兜底
async function runSearchOnly() {
  resetForRun();
  loading.value = true;
  try {
    statusText.value = "searching…";
    try {
      const res = await search(form.query, form.topK);
      hits.value = res?.hits || res?.items || [];
      statusText.value = `done: ${hits.value.length} hit(s)`;
    } catch {}
    try {
      const resAns = await answer(form.query, form.topK);
      const text = typeof resAns === 'string' ? resAns : (resAns?.text || resAns?.answer || JSON.stringify(resAns, null, 2));
      streamOut.value = text || "";
      pushMsg('assistant', text || "");
      stage.value = "done";
    } catch {}
  } catch (e) {
    error.value = e?.message || String(e);
  } finally {
    loading.value = false;
  }
}

// WS 流式
function connectWS() {
  try { ws?.close(); } catch {}
  const url = `${WS_BASE}/ws/answer`;
  streamOut.value = "";
  stage.value = "connecting";
  statusText.value = "connecting…";

  ws = new WebSocket(url);
  ws.onopen = () => {
    stage.value = "connected";
    statusText.value = "connected";
    ws.send(JSON.stringify({ type: "ask", query: form.query, top_k: form.topK }));
  };
  ws.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data);
      if (msg.type === "status") {
        if (msg.stage) stage.value = msg.stage;
        statusText.value = `[${msg.stage || 'status'}] ${msg.msg || ''}`.trim();
      }
      if (msg.type === "hits") hits.value = msg.items || [];
      if (msg.type === "delta") { streamOut.value += msg.text || ""; autoScroll(); }
      if (msg.type === "error") { error.value = msg.msg || "unknown error"; stage.value = "error"; }
      if (msg.type === "done") {
        stage.value = "done"; statusText.value = "done";
        pushMsg('assistant', streamOut.value);
        ws?.close();
      }
    } catch {
      stage.value = "llm";
      streamOut.value += ev.data;
      autoScroll();
    }
  };
  ws.onerror = () => { error.value = "ws error"; stage.value = "error"; };
  ws.onclose = () => {};
}

async function run() {
  if (!form.query.trim()) return;
  pushMsg('user', form.query);
  resetForRun();
  if (useStreaming.value) connectWS();
  else await runSearchOnly();
  form.query = "";
}
function stop() { try { ws?.close(); } catch {}; stage.value = "done"; }
function copyAnswer() {
  const txt = streamOut.value || (cur.value.messages.slice().reverse().find(m => m.role === 'assistant')?.content || "");
  navigator.clipboard.writeText(txt);
}
function newSession() {
  const id = 's-' + Date.now();
  sessions.value.unshift({ id, name: '新会话', messages: [] });
  currentSid.value = id;
}
onMounted(() => { if (!cur.value.messages.length) pushMsg('assistant', '你好，我是你的 AI 助手。'); });
onBeforeUnmount(() => { try { ws?.close(); } catch {} });

const isBusy = computed(() =>
  loading.value || ["connecting","connected","hf_bge","vector_search","llm"].includes(stage.value));
</script>

<template>
<div class="min-h-screen text-gray-900 bg-[radial-gradient(1200px_600px_at_-10%_-20%,#eef2ff,transparent),radial-gradient(1200px_600px_at_110%_-10%,#ecfeff,transparent)]">

<div id="layout"
  class="grid h-full gap-5 p-4 w-full max-w-[var(--page-max)] mx-auto
         grid-cols-1
         sm:grid-cols-[minmax(200px,240px)_minmax(0,1fr)]
         lg:grid-cols-[minmax(220px,280px)_minmax(0,1fr)_minmax(320px,380px)]
         2xl:grid-cols-[minmax(240px,320px)_minmax(0,1fr)_minmax(360px,440px)]">

      <aside class="hidden lg:block rounded-2xl border border-black/5 bg-white/60 backdrop-blur-xs shadow-soft p-4">
        <div class="flex items-center gap-2 mb-3">
          <div class="w-6 h-6 rounded-lg bg-[conic-gradient(from_0deg,#6366f1,#22d3ee,#a78bfa,#6366f1)]"></div>
          <div class="font-bold tracking-tight">RAG Studio</div>
        </div>
        <button class="w-full h-10 rounded-xl bg-gradient-to-tr from-[color:var(--color-brand-500)] to-cyan-400 text-white font-semibold shadow-[var(--shadow-card)]"
                @click="newSession">＋ 新会话</button>
        <div class="mt-3 space-y-2 max-h-[70vh] overflow-auto">
          <div v-for="s in sessions" :key="s.id"
               @click="currentSid=s.id"
               class="rounded-xl border border-gray-200 px-3 py-2 cursor-pointer bg-white hover:shadow-soft"
               :class="s.id===currentSid ? 'ring-2 ring-[color:var(--color-brand-400)]' : ''">
            <div class="font-semibold">{{ s.name }}</div>
            <div class="text-xs text-gray-500">{{ s.messages.filter(m=>m.role==='user').length }} 问题</div>
          </div>
        </div>
        <div class="mt-3 text-xs text-gray-500">WS: {{ stage }}</div>
      </aside>

      <!-- 中间 -->
      <section class="grid grid-rows-[auto_1fr_auto] gap-3 lg:col-span-2 2xl:col-span-2">
        <header class="rounded-2xl border border-black/5 bg-white/60 backdrop-blur-xs shadow-soft p-3 flex items-center justify-between">
          <div>
            <h1 class="text-base font-bold">对话</h1>
            <div v-if="statusText" class="text-xs text-gray-500">{{ statusText }}</div>
          </div>
          <StatusStepper :stage="stage"/>
        </header>

        <div ref="scrollRef" class="space-y-3 overflow-auto px-1">
          <template v-for="m in cur?.messages" :key="m.id">
            <div class="flex gap-3" :class="m.role==='user' ? 'justify-end' : ''">
              <div v-if="m.role!=='user'" class="w-7 h-7 rounded-full bg-[conic-gradient(from_0deg,#6366f1,#22d3ee,#a78bfa,#6366f1)]"></div>
              <div class="max-w-[min(760px,80%)] rounded-2xl border bg-white border-gray-200 shadow-soft px-4 py-3"
                   :class="m.role==='user' ? 'bg-gray-900 text-gray-100 border-transparent' : ''">
                {{ m.content }}
              </div>
              <div v-if="m.role==='user'" class="w-7 h-7 rounded-full bg-gray-900"></div>
            </div>
          </template>

          <!-- 流式 -->
          <div v-if="streamOut" class="flex gap-3">
            <div class="w-7 h-7 rounded-full bg-[conic-gradient(from_0deg,#6366f1,#22d3ee,#a78bfa,#6366f1)]"></div>
            <div class="max-w-[min(760px,80%)] rounded-2xl border bg-white border-gray-200 shadow-soft px-4 py-3">
              <span v-if="stage==='llm'" class="inline-block w-2.5 h-4 align-[-3px] mr-1 bg-[color:var(--color-brand-500)] animate-pulse"></span>
              {{ streamOut }}
            </div>
          </div>
        </div>

        <footer class="rounded-2xl border border-black/5 bg-white/70 backdrop-blur-xs shadow-soft p-3">
          <div class="grid grid-cols-[1fr_90px_auto_auto_auto] gap-2 items-center">
            <input class="h-11 rounded-xl border border-gray-200 bg-white px-4 outline-hidden focus:ring-4 focus:ring-[color:var(--color-brand-500)]/20"
                   placeholder="问点什么…" v-model="form.query" @keydown.enter.exact.prevent="run" :disabled="isBusy" />
            <input class="h-11 rounded-xl border border-gray-200 bg-white px-3 w-[90px]" type="number" min="1" max="20"
                   v-model.number="form.topK" title="Top-K" />
            <label class="text-sm text-gray-600 select-none">
              <input type="checkbox" v-model="useStreaming" class="align-middle mr-1"> WS 流式
            </label>
            <button class="h-11 rounded-xl bg-gray-900 text-white font-semibold px-4 disabled:opacity-60"
                    :disabled="isBusy || !form.query" @click="run">发送</button>
            <div class="flex gap-2">
              <button class="h-11 rounded-xl border border-gray-200 bg-white px-3 disabled:opacity-60" :disabled="!isBusy" @click="stop">停止</button>
              <button class="h-11 rounded-xl border border-gray-200 bg-white px-3 disabled:opacity-60"
                      :disabled="!(streamOut || cur?.messages?.length)" @click="copyAnswer">复制</button>
            </div>
          </div>
          <div v-if="error" class="mt-2 text-sm font-semibold text-red-600">⚠️ {{ error }}</div>
        </footer>
      </section>

      <!-- 右侧 -->
<!--      <aside class="hidden xl:block rounded-2xl border border-black/5 bg-white/60 backdrop-blur-xs shadow-soft p-4">-->
<!--        <div class="font-semibold mb-2">Top-K 检索</div>-->
<!--        <div v-if="!hits?.length" class="text-sm text-gray-500">暂无结果</div>-->
<!--        <ol v-else class="space-y-3 max-h-[75vh] overflow-auto">-->
<!--          <li v-for="(h,i) in hits" :key="h.id ?? i" class="border-b border-dashed border-gray-200 pb-3 last:border-0">-->
<!--            <div class="font-semibold">#{{ i+1 }} • {{ h.id ?? h.doc_id ?? '—' }}</div>-->
<!--            <div v-if="h.score != null" class="text-xs text-gray-500">score {{ Number(h.score).toFixed(3) }}</div>-->
<!--            <div v-if="h.text" class="mt-1 whitespace-pre-wrap text-sm">{{ h.text }}</div>-->
<!--            <details v-if="h.metadata" class="text-xs mt-1">-->
<!--              <summary>metadata</summary>-->
<!--              <pre class="whitespace-pre-wrap">{{ h.metadata }}</pre>-->
<!--            </details>-->
<!--          </li>-->
<!--        </ol>-->
<!--      </aside>-->
    </div>
  </div>
</template>

<style>
/* ======= 布局令牌（可按需改） ======= */
:root{
  --left-min: 220px;   --left-max: 300px;
  --right-min: 320px;  --right-max: 420px;
  --gap: 20px;
  --page-max: 1680px;  /* 想铺满超宽屏就删掉 max-width 相关 */
}

/* ======= 外层容器：高度/居中/背景撑满 ======= */
.min-h-screen { min-height: 100dvh !important; } /* 修正移动端 100vh 抖动 */

/* ======= 主网格：用我们自己的断点与列宽 ======= */
#layout {
  width: 100%;                      /* 替换 100vw */
  max-width: var(--page-max);       /* 恢复最大宽 */
  margin-inline: auto;              /* 恢复居中 */
  /* 其他不需要就别写，避免再覆盖 Tailwind */
}

/* ≥640px：两栏（左 + 中） */
@media (min-width: 640px){
  #layout{
    grid-template-columns:
      minmax(calc(var(--left-min) - 20px), calc(var(--left-max) - 40px))
      minmax(0,1fr) !important;
  }
}

/* ≥1024px：三栏（左 + 中 + 右） */
@media (min-width: 1024px){
  #layout{
    grid-template-columns:
      minmax(var(--left-min), var(--left-max))
      minmax(0,1fr)
      minmax(var(--right-min), var(--right-max)) !important;
  }
}

/* ≥1536px：适度再放大 */
@media (min-width: 1536px){
  #layout{
    grid-template-columns:
      minmax(calc(var(--left-min) + 20px), calc(var(--left-max) + 20px))
      minmax(0,1fr)
      minmax(calc(var(--right-min) + 20px), calc(var(--right-max) + 40px)) !important;
  }
}

/* ======= 关键修复：保证中列能“吃满” ======= */
/* 中间 section（聊天主体）不允许按内容撑出固有宽 */
#layout { min-height: 100%; }      /* 外层已是 min-h-screen 时，h-full 也可 */
#layout > section { min-width: 0; } /* 防止中列内容挤出布局 */

/* 聊天滚动区的容器也要能收缩，不然会挤压右栏或整体 */
#layout .overflow-auto { min-width: 0 !important; }

/* 让两侧面板顶部对齐（不被中部撑开） */


/* ======= 可选：顶部条吸顶（更像产品） ======= */
/*
#layout > section > header{
  position: sticky; top: 16px; z-index: 10;
}
*/

/* ======= 可选：去掉全局最大宽，真·全屏铺开 ======= */
/*
#layout{ max-width: none; }
*/
</style>
