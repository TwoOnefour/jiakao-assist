## 简介
无任何深度，~~得过且过~~

- python爬虫处理数据, 去重清洗生成bge向量
- 向量存入cloudflare ai向量数据库
- 使用bge RAG zh 处理向量转中文
- 最后将语义喂给llm最终生成结果

## 技术栈

### 前端
vite react 纯ai写的

### 后端
<img width="642" height="425" alt="image" src="https://github.com/user-attachments/assets/bcc21276-4215-4d71-8913-94e5a35b57e2" />

把llm和cloudflare拆成两个微服务解耦

cloudflare负责向量数据库存储和访问huggingface RAG完成向量-中文语义转换

llm负责根据已有的中文总结回复用户
