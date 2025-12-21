export interface Env {
  HF_TOKEN: string;
  HF_API_URL: string;
  HF_MODEL_DEFAULT: string;
  HF_MODEL?: string;
  VEC: VectorizeIndex;               // ← 来自 [[vectorize]] 绑定
}

type EmbedReq = { texts: string[], normalize?: boolean, model?: string, batch?: number };
type SearchReq = { query: string; topK?: number; filter?: Record<string, any>; normalize?: boolean };

const CORS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "content-type, authorization",
  "Access-Control-Allow-Methods": "POST, OPTIONS",
};

export default {
  async fetch(req: Request, env: Env): Promise<Response> {
    if (req.method === "OPTIONS") return new Response(null, { headers: CORS });

    const url = new URL(req.url);
	if (url.pathname === "/search" && req.method === "POST") {
	  const { query, topK = 5, filter, normalize = true } = await req.json() as SearchReq;
	  if (!query) return json({ error: "query required" }, 400);

	  const raw = (await callHF([query], env.HF_MODEL_DEFAULT, env))[0];
	  const vec = normalize ? l2(raw) : raw;  // 可能的归一化

	  const matches = await env.VEC.query(vec, {
		  topK,
		  returnValues: true,
		  returnMetadata: "all",
	  });

	  // 3) 整理返回
	  const hits = (matches?.matches ?? matches?.results ?? []).map((m: any) => ({
		id: m.id,
		score: m.score,
		metadata: m.metadata || {},
		text: m.metadata?.text,
	  }));
	  const resp = json({
		  code: 0,
		  message: {
			  data: { query, topK, hits }
		  },
	  }, 200)
	  return resp;
	}

    return new Response(JSON.stringify({msg: "胖次"}))
  }
} satisfies ExportedHandler<Env>;

async function callHF(texts: string[], model: string, env: Env): Promise<number[][]> {
  const url = `${env.HF_API_URL}/${encodeURIComponent(model)}`;
  const body = { inputs: texts, options: { wait_for_model: true } };

  // 简单重试（429/503）
  let lastErr: any;
  for (let attempt = 0; attempt < 3; attempt++) {
    const r = await fetch(url, {
      method: "POST",
      headers: {
        "Authorization": `Bearer ${env.HF_TOKEN}`,
        "Content-Type": "application/json"
      },
      body: JSON.stringify(body)
    });
    if (r.ok) {
      const data = await r.json();
      return ensureSentenceEmbeddings(data);
    }
    if (r.status === 429 || r.status === 503) {
      await sleep(1000 * (attempt + 1));
      lastErr = new Error(`HF ${r.status} ${r.statusText}`);
      continue;
    }
    const txt = await r.text();
    throw new Error(`HF error ${r.status}: ${txt}`);
  }
  throw lastErr || new Error("HF unknown error");
}

function ensureSentenceEmbeddings(ret: any): number[][] {
  // 情况A: [N][D]
  if (Array.isArray(ret) && Array.isArray(ret[0]) && typeof ret[0][0] === "number") {
    return ret as number[][];
  }
  // 情况B: [N][T][D] => mean-pooling
  if (Array.isArray(ret) && Array.isArray(ret[0]) && Array.isArray(ret[0][0])) {
    const arr: number[][][] = ret as number[][][];
    return arr.map(tokens => meanPool(tokens));
  }
  throw new Error("Unexpected HF response format");
}

function meanPool(tokens: number[][]): number[] {
  const D = tokens[0].length;
  const sums = new Array<number>(D).fill(0);
  for (const v of tokens) for (let i = 0; i < D; i++) sums[i] += v[i];
  for (let i = 0; i < D; i++) sums[i] /= tokens.length;
  return sums;
}

function l2(v: number[]): number[] {
  let s = 0; for (const x of v) s += x * x;
  const n = Math.sqrt(s) || 1;
  return v.map(x => x / n);
}

function json(obj: unknown, status = 200): Response {
  return new Response(JSON.stringify(obj), { status, headers: { "Content-Type": "application/json", ...CORS } });
}
const sleep = (ms: number) => new Promise(res => setTimeout(res, ms));
