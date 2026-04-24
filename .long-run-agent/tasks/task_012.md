# task_012

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_012.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

P02 LLMInvoker 接口 + 实现 — OpenAI/Claude/Ollama 调用、InvokeWithSchema

## 需求 (requirements)

- service/interfaces.go: LLMInvoker 接口定义
- plugins/llm/invoker.go: OpenAI/Claude/Ollama 实现
- Invoke: prompt + model + temperature → LLMResponse
- InvokeWithSchema: 结构化输出

## 验收标准 (acceptance)

- [ ] LLMInvoker 接口定义完整
- [ ] OpenAI Provider 实现
- [ ] Claude Provider 实现
- [ ] Ollama Provider 实现
- [ ] InvokeWithSchema 工作正常

## 交付物 (deliverables)

- `plugins/llm/invoker.go`
- `plugins/llm/provider_openai.go`
- `plugins/llm/provider_claude.go`
- `plugins/llm/provider_ollama.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 多 Provider 支持
- [ ] **测试验证**: API 调用测试
- [ ] **影响范围**: Eval 引擎、ModelAdapter

### 测试步骤
1. OpenAI 调用测试
2. Claude 调用测试
3. Ollama 调用测试
