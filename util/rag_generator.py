from pathlib import Path
import json, re, html

INPUT = Path("data/raw.jsonl")     # 源 JSONL
OUT_INSTRUCT = Path("data/train_instruct.jsonl")  # 指令/微调
OUT_RAG = Path("data/rag_corpus.jsonl")           # 检索语料
OUT_EVAL = Path("data/eval_set.jsonl")            # 评测集（仅问答对）

LETTERS = "ABCDEFGH"
HTML_SPACE_RE = re.compile(r"(?i)&nbsp;")

# ---------- 工具函数 ----------
def is_effectively_empty(s) -> bool:
    """更鲁棒的判空：处理 None、HTML 空格、全角空格、换行等。"""
    if s is None:
        return True
    s = str(s)
    s = HTML_SPACE_RE.sub(" ", s)         # &nbsp; -> space
    s = s.replace("\u3000", " ")          # 全角空格 -> space
    return s.strip() == ""

def present_from_options(options_texts):
    """根据 optionA~optionH 的文案是否为空，生成每一位是否“有效”的布尔列表。"""
    if not isinstance(options_texts, dict):
        return [True] * 8
    present = []
    for ch in LETTERS:
        txt = options_texts.get(f"option{ch}", "")
        present.append(not is_effectively_empty(txt))
    return present

def bitmask_to_letters(v, present, shift=0):
    """位掩码 -> 选项字母列表；可选整体右移 shift 位（默认不右移）。"""
    v = (v >> shift) & 0xFF
    return [LETTERS[i] for i in range(8) if (v & (1 << i)) and present[i]]

def decode_answer(ans, options_texts=None, allow_shift=True):
    """
    解码题库答案，兼容：
      - 字符串: 'B'、'AC'
      - 位掩码数值: A=1,B=2,C=4,D=8,E=16,F=32,G=64,H=128（必要时整体右移4位）
      - 序号数值: 1->A, 2->B,...
    右移规则：当且仅当 E–H 全空、低4位为0且高4位非0 时，整体右移4位。
    """
    if ans is None:
        return []

    present = present_from_options(options_texts)

    # 字符串
    if isinstance(ans, str):
        s = ans.strip().upper()
        return [ch for ch in s if ch in LETTERS and present[LETTERS.index(ch)]]

    # 数字
    if isinstance(ans, int):
        low4, high4 = (ans & 0x0F), (ans & 0xF0)

        # 🎯 优先：若满足偏移条件（E–H 全空、低4=0且高4!=0），先尝试右移4位解码
        if allow_shift and (not any(present[4:])) and low4 == 0 and high4 != 0:
            picked = bitmask_to_letters(ans, present, shift=4)
            if picked:
                return picked

        # 常规直接按位掩码
        picked = bitmask_to_letters(ans, present, shift=0)
        if picked:
            return picked

        # 兜底1：E–H 全空且低4位有值，只用低4位
        if not any(present[4:]) and low4:
            picked = bitmask_to_letters(low4, present, shift=0)
            if picked:
                return picked

        # 兜底2：序号编码
        if 1 <= ans <= 8:
            idx = ans - 1
            return [LETTERS[idx]] if present[idx] else []

    return []

def collect_options(obj):
    """提取存在文案的选项 (A-H)，保持顺序。"""
    opts = []
    for ch in LETTERS:
        key = f"option{ch}"
        raw = obj.get(key)
        if not is_effectively_empty(raw):
            opts.append((ch, str(raw).strip()))
    return opts

def clean_html(s):
    """清理解释里的 HTML，保留换行。"""
    if not s:
        return ""
    text = str(s)
    text = HTML_SPACE_RE.sub(" ", text)
    text = text.replace("\u3000", " ")
    text = re.sub(r"<br\s*/?>", "\n", text, flags=re.I)
    text = re.sub(r"<[^>]+>", "", text)  # 去标签
    text = html.unescape(text)           # 反转义
    return text.strip()

# ---------- 主流程 ----------
with INPUT.open("r", encoding="utf-8") as fin, \
     OUT_INSTRUCT.open("w", encoding="utf-8") as fo_ins, \
     OUT_RAG.open("w", encoding="utf-8") as fo_rag, \
     OUT_EVAL.open("w", encoding="utf-8") as fo_eval:

    for line in fin:
        line = line.strip()
        if not line:
            continue
        obj = json.loads(line)

        qid = obj.get("questionId") or obj.get("id") or ""
        question = (obj.get("question") or "").strip()
        options = collect_options(obj)

        # 关键：把整条 obj 传给 decode_answer，让它识别 E–H 是否为空
        answer_letters = decode_answer(obj.get("answer"), options_texts=obj, allow_shift=True)

        concise = clean_html(obj.get("conciseExplain"))
        explain = clean_html(obj.get("explain"))

        # 若仍解不出来，尝试用 assuredKeywords / concise / explain 做启发式回退（可选）
        if not answer_letters and options:
            fallback_sources = [obj.get("assuredKeywords"), concise, explain]
            joined = " ".join([s for s in fallback_sources if s])  # 拼接文本
            # 简单：若某个选项文案在解释里出现次数最多，则选它
            if joined:
                counts = {ch: joined.count(txt) for ch, txt in options if txt}
                if counts:
                    best = max(counts.items(), key=lambda kv: kv[1])
                    if best[1] > 0:
                        answer_letters = [best[0]]

        # —— 用途 A：指令/微调（通用“输入-输出”结构）——
        prompt = "请从以下选项中选择正确答案，并给出简要理由。\n"
        prompt += f"题目：{question}\n"
        for ch, txt in options:
            prompt += f"{ch}. {txt}\n"
        completion = {
            "answer_letters": answer_letters,  # 例如 ["B"]
            "answer_text": [txt for ch, txt in options if ch in set(answer_letters)],
            "explain": concise or explain
        }
        fo_ins.write(json.dumps({"input": prompt, "output": completion}, ensure_ascii=False) + "\n")

        # —— 用途 B：RAG 语料（可被向量化的纯文本+元数据）——
        rag_text = f"{question}\n" + "\n".join([f"{ch}. {txt}" for ch, txt in options])
        if concise or explain:
            rag_text += "\n解析：" + (concise or explain)
        rag_item = {
            "id": str(qid),
            "text": rag_text,
            "metadata": {
                "chapterId": obj.get("chapterId"),
                "label": obj.get("label"),
                "difficulty": obj.get("difficulty"),
                "keywords": obj.get("keywords")
            }
        }
        fo_rag.write(json.dumps(rag_item, ensure_ascii=False) + "\n")

        # —— 用途 C：评测集（问→标准答）——
        eval_item = {
            "id": str(qid),
            "question": question,
            "options": {ch: txt for ch, txt in options},
            "gold": answer_letters  # ["B"] 或多选如 ["A","C"]
        }
        fo_eval.write(json.dumps(eval_item, ensure_ascii=False) + "\n")
