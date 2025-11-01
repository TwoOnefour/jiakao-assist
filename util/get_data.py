#!/usr/bin/env python3
# -*- coding: utf-8 -*-
from requests import get
from time import time
from random import random
from find_root_path import find_project_root
from os import mkdir
from json import dumps

def gen_r(t: int = 1) -> (str, str):
    # n = abs(int(Date.now() * Math.random() * 1e4))
    n = abs(int(time() * 1000 * random() * 1e4))
    n_str = str(n)
    # o = sum(digits(n)) + len(n) ，再左补零到 3 位
    o = sum(int(ch) for ch in n_str) + len(n_str)
    o_str = str(o).rjust(3, "0")
    return f"{t}{n_str}{o_str}", f"0.{n}"

def build_url(key: str, t: int = 1) -> str:
    r, _ = gen_r(t)
    BASE = "https://api2.jiakaobaodian.com"
    PATH = "/api/web/exam-keyword/question-list.htm"
    return f"{BASE}{PATH}?_r={r}&key={key}&_={_}"

def fetch(key="4ae269de"):
    url = build_url(key)
    headers = {
        "Referer": "https://www.jiakaobaodian.com/",
        "Origin": "https://www.jiakaobaodian.com",
        "Accept": "application/json, text/plain, */*",
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126 Safari/537.36",
    }
    # print("GET", url)
    resp = get(url, headers=headers, timeout=10)
    # print("status:", resp.status_code)
    # print(resp.text[:2000])  # 预览前 2000 字符
    return resp.json()["data"]

def save(data):
    ROOT = find_project_root()
    DATA_DIR = ROOT / 'data'
    if not DATA_DIR.exists():
        mkdir(DATA_DIR)

    with open(f'{DATA_DIR}/raw.jsonl', 'w', encoding='utf-8') as f:
        for d in data:
            f.write(dumps(d.json, ensure_ascii=False))
            f.write('\n')


class Data:
    """
    自定义data哈希键比较，key为str(json['questionId']) + str(json['id']) + json['question']，减少哈希碰撞
    """
    def __init__(self, json):
        self.json = json
        self.key = str(json['questionId']) + str(json['id']) + json['question']

    def __hash__(self):
        return hash(self.key)

    def __eq__(self, other):
        return isinstance(other, Data) and self.key == other.key

if __name__ == "__main__":
    question_id_set = set()
    data = []
    # O（n）, 可以直接开4协程跟他报了
    async_list = []
    for i in range(100):
        try:
            cur_data = fetch()
        except Exception:
            continue
        data += cur_data # O(1)

    data = set(Data(_data) for _data in data) # O（n）
    # 去重后也就150条，说好的1000题呢？
    save(data)
