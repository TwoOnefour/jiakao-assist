## 简介
一个llm驾校考试小助手

示例网页： https://assist.0rtt.de 

无任何深度，~~得过且过~~

- python爬虫处理数据, 去重清洗生成bge向量
- 向量存入cloudflare ai向量数据库
- 使用bge RAG zh 处理向量转中文
- 最后将语义喂给llm最终生成结果

## 技术栈

### 前端
vite react 纯ai写的

### 后端
<img width="2816" height="1536" alt="Gemini_Generated_Image_mzx48smzx48smzx4" src="https://github.com/user-attachments/assets/d7d9a272-511e-46f5-b6bb-8e7b8e79f61c" />

把llm和cloudflare拆成两个微服务解耦

cloudflare负责向量数据库存储和访问huggingface RAG完成向量-中文语义转换

llm负责根据已有的中文总结回复用户
