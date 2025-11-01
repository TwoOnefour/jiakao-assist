#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import os, json, argparse
from pathlib import Path
from tqdm import tqdm
import chromadb
from sentence_transformers import SentenceTransformer

def read_jsonl(path: Path):
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line=line.strip()
            if line:
                yield json.loads(line)

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--corpus", default="data/rag_corpus.jsonl", help="JSONL 语料路径")
    ap.add_argument("--index_dir", default="index/chroma", help="Chroma 持久化路径")
    ap.add_argument("--collection", default="jiakao", help="集合名")
    ap.add_argument("--model", default="BAAI/bge-small-zh-v1.5", help="bge 模型名")
    ap.add_argument("--batch", type=int, default=128, help="编码 batch size")
    ap.add_argument("--normalize", action="store_true", help="是否归一化向量（cosine 推荐）")
    args = ap.parse_args()

    corpus = Path(args.corpus)
    if not corpus.exists():
        raise SystemExit(f"找不到语料文件：{corpus}")

    print(f"加载模型：{args.model}")
    model = SentenceTransformer(args.model)

    print(f"初始化 Chroma 持久化目录：{args.index_dir}")
    client = chromadb.PersistentClient(path=args.index_dir)
    coll = client.get_or_create_collection(args.collection, metadata={"hnsw:space":"cosine"})

    # 读取全部文档
    docs, ids, metas = [], [], []
    for row in read_jsonl(corpus):
        _id  = str(row["id"])
        text = str(row["text"])
        meta = row.get("metadata", {})
        if not text or not _id:
            continue
        ids.append(_id)
        docs.append(text)
        metas.append(meta)

    print(f"待索引文档数：{len(docs)}")
    if not docs:
        return

    # 分批编码 + upsert
    B = args.batch
    for i in tqdm(range(0, len(docs), B), desc="embedding & upsert"):
        seg_docs  = docs[i:i+B]
        seg_ids   = ids[i:i+B]
        seg_metas = metas[i:i+B]
        embs = model.encode(seg_docs, batch_size=B, show_progress_bar=False,
                            normalize_embeddings=args.normalize)
        coll.upsert(ids=seg_ids, documents=seg_docs,
                    embeddings=embs.tolist(), metadatas=seg_metas)

    print("✅ 索引完成")
    print(f"集合 {args.collection} 文档数：", coll.count())

if __name__ == "__main__":
    main()
