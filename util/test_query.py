#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import argparse
import chromadb
from sentence_transformers import SentenceTransformer

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--index_dir", default="index/chroma")
    ap.add_argument("--collection", default="jiakao")
    ap.add_argument("--model", default="BAAI/bge-small-zh-v1.5")
    ap.add_argument("--query", default="会车遇到障碍时谁先行？")
    ap.add_argument("--top_k", type=int, default=5)
    ap.add_argument("--normalize", action="store_true")
    args = ap.parse_args()

    client = chromadb.PersistentClient(path=args.index_dir)
    coll = client.get_or_create_collection(args.collection)
    embedder = SentenceTransformer(args.model)

    q_emb = embedder.encode([args.query], normalize_embeddings=args.normalize)
    res = coll.query(query_embeddings=q_emb, n_results=args.top_k)
    docs = res["documents"][0]
    metas= res["metadatas"][0]
    ids  = res["ids"][0]

    print(f"Query: {args.query}\nTop-{args.top_k} 命中：\n")
    for i, (doc, meta, id_) in enumerate(zip(docs, metas, ids), start=1):
        print(f"[{i}] id={id_} topic={meta.get('topic_label')}")
        print(doc[:200].replace("\n","  "), "...\n")

if __name__ == "__main__":
    main()