## 简介
一个llm驾校考试小助手

示例网页： https://assist.0rtt.de 

无任何深度，~~得过且过~~，架构图见下面

- python爬虫处理数据, 去重清洗生成bge向量
- 向量存入cloudflare ai向量数据库
- 使用bge RAG zh 处理向量转中文
- 最后将语义喂给llm最终生成结果

## 项目结构
- util python 爬虫、数据清理、向量生成
- vite-project 前端
- go-api-backend 后端
- hf-worker cloudflare worker的无状态边缘函数

## 技术栈

### 前端
vite react 纯ai写的

### 后端
自行看图吧，不要问我如果访问量高了怎么办，什么是削峰和队列解耦缓存问题，这些都没有

<img width="2816" height="1536" alt="Gemini_Generated_Image_mzx48smzx48smzx4" src="https://github.com/user-attachments/assets/d7d9a272-511e-46f5-b6bb-8e7b8e79f61c" />

把llm和cloudflare拆成两个微服务解耦

cloudflare负责向量数据库存储和访问huggingface RAG完成向量-中文语义转换

llm负责根据已有的中文总结回复用户

## Disclaimer
数据源提供：https://www.jiakaobaodian.com/kaoshi/4ae269de.html

请不要滥用爬虫
