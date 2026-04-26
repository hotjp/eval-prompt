import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Input, Button, Space, message, Spin, Tabs, Tag, Modal, Select } from 'antd'
import { SaveOutlined, PlayCircleOutlined, DiffOutlined, EditOutlined, SendOutlined, ClearOutlined, PlusOutlined, DeleteOutlined, SwapOutlined } from '@ant-design/icons'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import MonacoEditor, { DiffEditor } from '@monaco-editor/react'
import { assetApi, triggerApi, llmApi } from '../api/client'
import type { AssetDetail } from '../api/client'
import { getLLMConfigs } from '../config/llmConfig'
import './EditorViewV2.css'

function formatUpdatedAt(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  return `${diffDays}d ago`
}

const defaultBody = `## Instruction

Write your prompt here.

## Examples

### Example 1
- Input:
- Output:

## Variables

- \`{{variable_name}}\`: description
`

interface Draft {
  content: string
  savedHash: string
  savedAt: number
}

interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: number
}

function getDraftKey(id: string) {
  return `draft:${id}`
}

function getVariablesKey(id: string) {
  return `variables:${id}`
}

function loadVariables(id: string): Array<{ key: string; value: string }> | null {
  try {
    const raw = localStorage.getItem(getVariablesKey(id))
    if (!raw) return null
    return JSON.parse(raw)
  } catch {
    return null
  }
}

function saveVariables(id: string, vars: Array<{ key: string; value: string }>) {
  localStorage.setItem(getVariablesKey(id), JSON.stringify(vars))
}

function loadDraft(id: string): Draft | null {
  try {
    const raw = localStorage.getItem(getDraftKey(id))
    if (!raw) return null
    const draft: Draft = JSON.parse(raw)
    if (Date.now() - draft.savedAt > 7 * 24 * 60 * 60 * 1000) {
      localStorage.removeItem(getDraftKey(id))
      return null
    }
    return draft
  } catch {
    return null
  }
}

function saveDraft(id: string, content: string, hash: string) {
  const draft: Draft = { content, savedHash: hash, savedAt: Date.now() }
  localStorage.setItem(getDraftKey(id), JSON.stringify(draft))
}

function clearDraft(id: string) {
  localStorage.removeItem(getDraftKey(id))
}

function generateId() {
  return Math.random().toString(36).substring(2, 11)
}

function stripThinkTags(content: string): string {
  return content.replace(/<think>[\s\S]*?<\/think>/g, '')
}

function EditorViewV2() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [promptValue, setPromptValue] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [hasChanges, setHasChanges] = useState(false)
  const [injectedResult, setInjectedResult] = useState('')
  const [validating, setValidating] = useState(false)
  const [contentHash, setContentHash] = useState('')
  const [updatedAt, setUpdatedAt] = useState('')
  const [conflictDraft, setConflictDraft] = useState<Draft | null>(null)
  const [conflictServerContent, setConflictServerContent] = useState('')
  const [showRewriteInput, setShowRewriteInput] = useState(false)
  const [rewriteInstruction, setRewriteInstruction] = useState('')
  const [rewriting, setRewriting] = useState(false)
  const [rewritePreview, setRewritePreview] = useState('')
  const [showRewritePreview, setShowRewritePreview] = useState(false)
  const [activeTab, setActiveTab] = useState('editor')
  const loadedRef = useRef(false)
  const autoSaveTimer = useRef<number | undefined>(undefined)
  const chatMessagesEndRef = useRef<HTMLDivElement>(null)

  // Chat state
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([])
  const [chatInput, setChatInput] = useState('')
  const [chatLoading, setChatLoading] = useState(false)
  const llmConfigs = getLLMConfigs()
  const [selectedModel, setSelectedModel] = useState<string>(llmConfigs.find(c => c.default_model)?.name || llmConfigs[0]?.name || '')

  // Variables state
  const [variables, setVariables] = useState<Array<{ key: string; value: string }>>(() => {
    if (id && id !== 'new') {
      const saved = loadVariables(id)
      if (saved && saved.length > 0) return saved
    }
    return [{ key: '', value: '' }]
  })

  // Save variables to localStorage when they change
  useEffect(() => {
    if (id && id !== 'new') {
      saveVariables(id, variables)
    }
  }, [variables, id])

  useEffect(() => {
    chatMessagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [chatMessages])

  useEffect(() => {
    if (!id || id === 'new' || !hasChanges) return
    clearTimeout(autoSaveTimer.current)
    autoSaveTimer.current = setTimeout(() => {
      saveDraft(id, promptValue, contentHash)
    }, 2000)
    return () => clearTimeout(autoSaveTimer.current)
  }, [promptValue, contentHash, hasChanges, id])

  useEffect(() => {
    if (!id || id === 'new') {
      setLoading(false)
      return
    }

    if (loadedRef.current) return

    let timeout: number
    let cancelled = false

    const loadAsset = async () => {
      setLoading(true)
      try {
        const race = Promise.race([
          Promise.all([
            assetApi.get(id).catch(() => null),
            assetApi.getContent(id).catch(() => ({ content: '', content_hash: '', updated_at: '' })),
          ]),
          new Promise<null>((_, reject) => {
            timeout = setTimeout(() => reject(new Error('timeout')), 3000)
          }),
        ])

        const [assetData, contentData] = await race as [AssetDetail | null, { content: string; content_hash: string; updated_at: string }]

        if (cancelled) return
        clearTimeout(timeout)

        if (!assetData) {
          message.error('Asset not found')
          navigate('/assets')
          return
        }

        setAsset(assetData)

        const draft = loadDraft(id)
        const serverContent = contentData.content || defaultBody
        const serverHash = contentData.content_hash || ''

        if (draft && draft.savedHash && draft.savedHash !== serverHash) {
          setConflictDraft(draft)
          setConflictServerContent(serverContent)
          setPromptValue(draft.content)
          setOriginalContent(draft.content)
        } else {
          const content = (draft && !serverContent) ? draft.content : (contentData.content || defaultBody)
          setPromptValue(content)
          setOriginalContent(content)
          if (draft) saveDraft(id, content, serverHash)
        }

        setContentHash(serverHash)
        setUpdatedAt(contentData.updated_at || '')
        loadedRef.current = true
      } catch (err) {
        if (cancelled) return
        message.error('Failed to load asset')
        console.error(err)
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    loadAsset()

    return () => {
      cancelled = true
      clearTimeout(timeout)
    }
  }, [id])

  useEffect(() => {
    setHasChanges(promptValue !== originalContent)
  }, [promptValue, originalContent])

  const handleSave = async (forceContent?: string) => {
    if (!id) return
    setSaving(true)
    try {
      const result = await assetApi.saveContent(id, forceContent || promptValue, undefined, contentHash)
      message.success(result.message || 'Saved')
      setPromptValue(result.content)
      setOriginalContent(result.content)
      setContentHash(result.content_hash)
      setUpdatedAt(result.updated_at)
      setHasChanges(false)
      clearDraft(id)
    } catch (err: any) {
      if (err?.response?.status === 409) {
        message.warning('Conflict detected. Please choose which version to keep.')
        try {
          const contentData = await assetApi.getContent(id)
          const draft = loadDraft(id)
          setConflictDraft(draft)
          setConflictServerContent(contentData.content || defaultBody)
          setContentHash(contentData.content_hash || '')
          setUpdatedAt(contentData.updated_at || '')
        } catch {
          message.error('Failed to reload content')
        }
      } else {
        message.error('Failed to save')
      }
    } finally {
      setSaving(false)
    }
  }

  const handleCommit = async () => {
    if (!id) return
    setSaving(true)
    try {
      const result = await assetApi.commit(id, `Update prompt ${id}`)
      message.success('Committed: ' + result.commit.slice(0, 8))
    } catch {
      message.error('Failed to commit')
    } finally {
      setSaving(false)
    }
  }

  const handleAcceptServer = () => {
    if (!conflictServerContent) return
    clearDraft(id!)
    setConflictDraft(null)
    setConflictServerContent('')
    setPromptValue(conflictServerContent)
    setOriginalContent(conflictServerContent)
    setHasChanges(false)
  }

  const handleKeepLocal = () => {
    if (!conflictDraft) return
    setConflictDraft(null)
    setConflictServerContent('')
    setHasChanges(true)
  }

  const handleValidate = async () => {
    setValidating(true)
    try {
      const result = await triggerApi.validate(promptValue)
      if (result.valid) {
        message.success('Prompt is valid')
      } else {
        message.warning(result.message || 'Invalid prompt')
      }
    } catch {
      message.error('Validation failed')
    } finally {
      setValidating(false)
    }
  }

  const handleInject = async () => {
    try {
      const vars: Record<string, string> = {}
      variables.forEach(({ key, value }) => {
        if (key.trim()) {
          vars[key.trim()] = value
        }
      })
      const result = await triggerApi.inject(promptValue, vars)
      setInjectedResult(result.result)
      setActiveTab('preview')
    } catch {
      message.error('Failed to inject variables')
    }
  }

  const addVariable = () => {
    setVariables([...variables, { key: '', value: '' }])
  }

  const removeVariable = (index: number) => {
    setVariables(variables.filter((_, i) => i !== index))
  }

  const updateVariable = (index: number, field: 'key' | 'value', val: string) => {
    const updated = [...variables]
    updated[index][field] = val
    setVariables(updated)
  }

  const handleRestore = () => {
    setPromptValue(originalContent)
    setHasChanges(false)
  }

  const handleRewrite = async () => {
    if (!rewriteInstruction.trim()) {
      message.warning('Please enter a rewrite instruction')
      return
    }
    setRewriting(true)
    try {
      const result = await llmApi.rewrite(promptValue, rewriteInstruction, selectedModel, true)
      setRewritePreview(result.rewritten)
      setShowRewritePreview(true)
    } catch (err: any) {
      if (err?.response?.status === 503) {
        message.warning('请先在设置中配置 LLM')
      } else {
        message.error('Rewrite failed')
      }
    } finally {
      setRewriting(false)
    }
  }

  const handleApplyRewrite = () => {
    setPromptValue(rewritePreview)
    setShowRewritePreview(false)
    setShowRewriteInput(false)
    setRewriteInstruction('')
    setRewritePreview('')
    message.success('Rewrite applied')
  }

  const handleCancelRewrite = () => {
    setShowRewritePreview(false)
    setRewritePreview('')
  }

  const handleChatSend = async () => {
    if (!chatInput.trim() || chatLoading) return

    const userMessage: ChatMessage = {
      id: generateId(),
      role: 'user',
      content: chatInput,
      timestamp: Date.now(),
    }

    setChatMessages(prev => [...prev, userMessage])
    setChatInput('')
    setChatLoading(true)

    try {
      const result = await llmApi.chat(chatInput, selectedModel)
      const assistantMessage: ChatMessage = {
        id: generateId(),
        role: 'assistant',
        content: result.content || '响应完成',
        timestamp: Date.now(),
      }
      setChatMessages(prev => [...prev, assistantMessage])
    } catch (err: any) {
      const errorMessage: ChatMessage = {
        id: generateId(),
        role: 'assistant',
        content: err?.response?.status === 503 ? '请先在设置中配置 LLM' : '请求失败',
        timestamp: Date.now(),
      }
      setChatMessages(prev => [...prev, errorMessage])
    } finally {
      setChatLoading(false)
    }
  }

  const handleClearChat = () => {
    setChatMessages([])
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  if (!asset && id !== 'new') {
    return <div>Asset not found</div>
  }

  const category = asset?.category || 'content'
  if (category !== 'content') {
    return (
      <Card title={asset?.name || id}>
        <Space direction="vertical">
          <Tag color="blue">{category}</Tag>
          <p>This asset type does not use the markdown editor.</p>
          <p>For <strong>eval</strong> assets, edit test cases in the Cases tab.</p>
          <p>For <strong>metric</strong> assets, edit rubrics in the Rubric tab.</p>
          <Button onClick={() => navigate(`/assets/${id}`)}>Go to Detail View</Button>
        </Space>
      </Card>
    )
  }

  return (
    <div className="editor-v2-container">
      {/* Left Panel - Editor */}
      <div className="editor-panel">
        {/* Header */}
        <div className="editor-header">
          <div className="editor-header-left">
            <span className="editor-title">{asset?.name || id}</span>
            {asset?.asset_type && <Tag className="editor-tag">{asset.asset_type}</Tag>}
            {asset?.state && <Tag color={asset.state === 'active' ? 'green' : 'orange'}>{asset.state}</Tag>}
            {updatedAt && <Tag color="blue" className="editor-time-tag">Saved {formatUpdatedAt(updatedAt)}</Tag>}
          </div>
          <div className="editor-header-nav">
            <Button
              size="small"
              onClick={() => {
                if (hasChanges && !window.confirm('You have unsaved changes. Leave anyway?')) return
                navigate(`/assets/${id}/versions`)
              }}
            >
              Version Tree
            </Button>
            <Button
              size="small"
              onClick={() => {
                if (hasChanges && !window.confirm('You have unsaved changes. Leave anyway?')) return
                navigate(`/assets/${id}/eval`)
              }}
            >
              Run Eval
            </Button>
            <Button
              size="small"
              icon={<SwapOutlined />}
              onClick={() => {
                if (hasChanges && !window.confirm('You have unsaved changes. Leave anyway?')) return
                navigate('/compare')
              }}
            >
              Compare
            </Button>
          </div>
        </div>

        {/* Toolbar */}
        <div className="editor-toolbar">
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            className="editor-tabs"
            items={[
              { key: 'editor', label: 'Editor' },
              { key: 'diff', label: <span><DiffOutlined /> Diff</span> },
              { key: 'preview', label: 'Preview' },
            ]}
          />
          <div className="toolbar-actions">
            {showRewriteInput ? (
              <>
                <Input
                  placeholder="改写指令..."
                  value={rewriteInstruction}
                  onChange={(e) => setRewriteInstruction(e.target.value)}
                  onPressEnter={handleRewrite}
                  style={{ width: 160 }}
                  size="small"
                  disabled={rewriting}
                />
                <Button size="small" type="primary" onClick={handleRewrite} loading={rewriting}>Apply</Button>
                <Button size="small" onClick={() => { setShowRewriteInput(false); setRewriteInstruction('') }} disabled={rewriting}>Cancel</Button>
              </>
            ) : (
              <>
                <Button icon={<PlayCircleOutlined />} onClick={handleValidate} loading={validating} size="small">
                  Validate
                </Button>
                <Button size="small" icon={<EditOutlined />} onClick={() => setShowRewriteInput(true)}>
                  Rewrite
                </Button>
                {hasChanges && (
                  <Button size="small" onClick={handleRestore}>
                    Restore
                  </Button>
                )}
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  onClick={() => handleSave()}
                  loading={saving}
                  disabled={!hasChanges}
                  size="small"
                >
                  Save
                </Button>
                <Button icon={<SaveOutlined />} onClick={handleCommit} loading={saving} size="small">
                  Commit
                </Button>
              </>
            )}
          </div>
        </div>

        {/* Monaco Editor */}
        <div className="editor-content">
          {activeTab === 'editor' && (
            <MonacoEditor
              height="100%"
              language="markdown"
              value={promptValue}
              onChange={(value) => setPromptValue(value || '')}
              theme="vs-dark"
              options={{ minimap: { enabled: false } }}
            />
          )}
          {activeTab === 'diff' && (
            <div className="diff-container">
              {hasChanges ? (
                <DiffEditor
                  height="100%"
                  language="markdown"
                  original={originalContent}
                  modified={promptValue}
                  theme="vs-dark"
                  options={{
                    minimap: { enabled: false },
                    renderSideBySide: true,
                    readOnly: true,
                  }}
                />
              ) : (
                <div className="no-changes">No changes to show</div>
              )}
            </div>
          )}
          {activeTab === 'preview' && (
            <div className="preview-container markdown-body">
              {injectedResult ? (
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{injectedResult}</ReactMarkdown>
              ) : (
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{promptValue}</ReactMarkdown>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Right Panel - Chat */}
      <div className="chat-panel">
        {/* Chat Header */}
        <div className="chat-header">
          <span className="chat-title">AI Assistant</span>
          <div className="chat-header-actions">
            <Select
              value={selectedModel}
              onChange={setSelectedModel}
              size="small"
              style={{ width: 150 }}
              options={llmConfigs.map(c => ({
                value: c.name,
                label: `${c.name} - ${c.default_model}`,
              }))}
              placeholder="Select provider"
            />
            <Button icon={<ClearOutlined />} size="small" onClick={handleClearChat} />
          </div>
        </div>

        {/* Chat Messages */}
        <div className="chat-messages">
          {chatMessages.length === 0 && (
            <div className="chat-empty">
              <p>发送消息开始与 AI 对话</p>
              <p className="chat-empty-hint">可以要求 AI 改写、优化或解释当前 Prompt</p>
            </div>
          )}
          {chatMessages.map((msg) => (
            <div key={msg.id} className={`chat-message ${msg.role}`}>
              <div className="message-bubble">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{stripThinkTags(msg.content)}</ReactMarkdown>
              </div>
            </div>
          ))}
          {chatLoading && (
            <div className="chat-message assistant">
              <div className="message-bubble loading">
                <Spin size="small" /> AI 思考中...
              </div>
            </div>
          )}
          <div ref={chatMessagesEndRef} />
        </div>

        {/* Chat Input */}
        <div className="chat-input-container">
          <Input
            value={chatInput}
            onChange={(e) => setChatInput(e.target.value)}
            onPressEnter={handleChatSend}
            placeholder="输入消息..."
            className="chat-input"
          />
          <Button
            type="primary"
            icon={<SendOutlined />}
            onClick={handleChatSend}
            loading={chatLoading}
            disabled={!chatInput.trim()}
          />
        </div>

        {/* Inject Section */}
        <div className="inject-section">
          <div className="inject-header">
            <span className="inject-title">Variables</span>
            <Button size="small" icon={<PlusOutlined />} onClick={addVariable}>Add</Button>
          </div>
          <div className="inject-variables">
            {variables.map((v, index) => (
              <div key={index} className="variable-row">
                <Input
                  placeholder="Key"
                  value={v.key}
                  onChange={(e) => updateVariable(index, 'key', e.target.value)}
                  className="variable-key"
                />
                <span className="variable-separator">:</span>
                <Input
                  placeholder="Value"
                  value={v.value}
                  onChange={(e) => updateVariable(index, 'value', e.target.value)}
                  className="variable-value"
                />
                <Button
                  size="small"
                  icon={<DeleteOutlined />}
                  onClick={() => removeVariable(index)}
                  disabled={variables.length === 1}
                />
              </div>
            ))}
          </div>
          <Button type="primary" onClick={handleInject} block>
            Inject
          </Button>
        </div>
      </div>

      {/* Conflict Modal */}
      <Modal
        title="Conflict Detected"
        open={!!conflictDraft}
        width={900}
        footer={null}
        onCancel={() => setConflictDraft(null)}
      >
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <p>Your local draft differs from the server version. Choose which to keep:</p>
          <DiffEditor
            height={300}
            language="markdown"
            original={conflictServerContent || 'Server version'}
            modified={conflictDraft?.content || ''}
            theme="vs-dark"
            options={{ minimap: { enabled: false }, renderSideBySide: true }}
          />
          <Space>
            <Button type="primary" onClick={() => handleSave(conflictDraft?.content)}>
              Keep Local Draft
            </Button>
            <Button onClick={handleAcceptServer}>
              Accept Server Version
            </Button>
            <Button onClick={handleKeepLocal}>
              Review &amp; Merge Later
            </Button>
          </Space>
        </Space>
      </Modal>

      {/* Rewrite Preview Modal */}
      <Modal
        title="Rewrite Preview"
        open={showRewritePreview}
        width={900}
        onCancel={handleCancelRewrite}
        footer={[
          <Button key="cancel" onClick={handleCancelRewrite}>
            Cancel
          </Button>,
          <Button key="apply" type="primary" onClick={handleApplyRewrite}>
            Apply Rewrite
          </Button>,
        ]}
      >
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <p>Review the rewritten content below:</p>
          <DiffEditor
            height={400}
            language="markdown"
            original={promptValue}
            modified={rewritePreview}
            theme="vs-dark"
            options={{ minimap: { enabled: false }, renderSideBySide: true }}
          />
        </Space>
      </Modal>
    </div>
  )
}

export default EditorViewV2
