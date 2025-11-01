const API_BASE = import.meta.env.VITE_API_BASE;

export async function search(query, topK = 5) {
  const res = await fetch(`${API_BASE}/search`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, top_k: topK })
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const data = await res.json();

  // 兼容你的 CommonResp 或直出
  if (data?.message?.data) return data.message.data;
  return data;
}

// 如果你已经有 /answer（非流式），可以用它；没有就先不用
export async function answer(query, topK = 5) {
  const res = await fetch(`${API_BASE}/answer`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, top_k: topK })
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const data = await res.json();
  if (data?.message?.data) return data.message.data;
  return data;
}
