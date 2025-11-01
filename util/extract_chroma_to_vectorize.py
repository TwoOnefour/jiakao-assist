# export_chroma_to_vectorize.py
import json, chromadb, os, requests

INDEX_DIR = "index/chroma"
COLL_NAME = "jiakao"               # 与 build_index.py 一致
NDJSON = "data\embeddings.ndjson"

client = chromadb.PersistentClient(path=INDEX_DIR)
coll = client.get_collection(COLL_NAME)

# 一次性取完；很大时可用 limit/offset 分批
data = coll.get(include=["documents", "metadatas", "embeddings"])
with open(NDJSON, "w", encoding="utf-8") as f:
    for _id, doc, emb, meta in zip(data["ids"], data["documents"], data["embeddings"], data["metadatas"]):
        item = {"id": str(_id), "values": list(map(float, emb)), "metadata": {**(meta or {}), "text": doc}}
        f.write(json.dumps(item, ensure_ascii=False) + "\n")

print("NDJSON written:", NDJSON)
