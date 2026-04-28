import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Input, Button, Space, message, Spin, Tabs, Tag, Modal, Select } from 'antd'
import { SaveOutlined, PlayCircleOutlined, DiffOutlined, EditOutlined, SendOutlined, ClearOutlined, PlusOutlined, DeleteOutlined, SwapOutlined, LoadingOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { loader } from '@monaco-editor/react'
import MonacoEditor, { DiffEditor } from '@monaco-editor/react'
import { assetApi, triggerApi, llmApi } from '../api/client'
import QuickEvalModal from './eval/components/QuickEvalModal'
import type { AssetDetail } from '../api/client'
import { getLLMConfigs } from '../config/llmConfig'
import './EditorViewV2.css'

// Configure Monaco to use local files instead of CDN
loader.config({ paths: { vs: '/monaco-editor/min' } })

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

function extractThinkContent(content: string): { think: string; clean: string } {
  const thinkMatches = content.match(/<think>[\s\S]*?<\/think>/g)
  const think = thinkMatches ? thinkMatches.join('\n').replace(/<\/?think>/g, '') : ''
  const clean = content.replace(/<think>[\s\S]*?<\/think>/g, '')
  return { think, clean }
}

function EditorViewV2() {
  const { t } = useTranslation()
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
  const [showQuickEval, setShowQuickEval] = useState(false)
  const [activeTab, setActiveTab] = useState('editor')
  const loadedRef = useRef(false)
  const autoSaveTimer = useRef<number | undefined>(undefined)
  const chatMessagesEndRef = useRef<HTMLDivElement>(null)

  // Chat state
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([])
  const [expandedThinks, setExpandedThinks] = useState<Set<string>>(new Set())
  const [chatInput, setChatInput] = useState('')
  const [chatLoading, setChatLoading] = useState(false)
  const llmConfigs = getLLMConfigs()
  const [selectedModel, setSelectedModel] = useState<string>(llmConfigs.find(c => c.default_model)?.name || llmConfigs[0]?.name || '')
  const [monacoReady, setMonacoReady] = useState(false)

  const toggleThink = (id: string) => {
    setExpandedThinks(prev => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

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
          message.error(t('editor_asset_not_found'))
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
        message.error(t('editor_load_failed'))
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
      message.success(result.message || t('editor_saved'))
      setPromptValue(result.content)
      setOriginalContent(result.content)
      setContentHash(result.content_hash)
      setUpdatedAt(result.updated_at)
      setHasChanges(false)
      clearDraft(id)
    } catch (err: any) {
      if (err?.response?.status === 409) {
        message.warning(t('editor_conflict_detected'))
        try {
          const contentData = await assetApi.getContent(id)
          const draft = loadDraft(id)
          setConflictDraft(draft)
          setConflictServerContent(contentData.content || defaultBody)
          setContentHash(contentData.content_hash || '')
          setUpdatedAt(contentData.updated_at || '')
        } catch {
          message.error(t('editor_reload_failed'))
        }
      } else {
        message.error(t('editor_save_failed'))
      }
    } finally {
      setSaving(false)
    }
  }

  const handleCommit = async () => {
    if (!id) return
    setSaving(true)
    try {
      // Save first if there are unsaved changes
      if (hasChanges) {
        const result = await assetApi.saveContent(id, promptValue, undefined, contentHash)
        setPromptValue(result.content)
        setOriginalContent(result.content)
        setContentHash(result.content_hash)
        setUpdatedAt(result.updated_at)
        setHasChanges(false)
        clearDraft(id)
      }
      // Then commit
      const result = await assetApi.commit(id, `Update prompt ${id}`)
      message.success(t('editor_commit_success') + result.commit.slice(0, 8))
    } catch {
      message.error(t('editor_commit_failed'))
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
        message.success(t('editor_valid'))
      } else {
        message.warning(result.message || t('editor_invalid'))
      }
    } catch {
      message.error(t('editor_validation_failed'))
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
      message.error(t('editor_inject_failed'))
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
      message.warning(t('editor_rewrite_placeholder'))
      return
    }
    setRewriting(true)
    try {
      const result = await llmApi.rewrite(promptValue, rewriteInstruction, selectedModel, true)
      setRewritePreview(result.rewritten)
      setShowRewritePreview(true)
    } catch (err: any) {
      if (err?.response?.status === 503) {
        message.warning(t('editor_rewrite_configure_llm'))
      } else {
        message.error(t('editor_rewrite_failed'))
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
    message.success(t('editor_rewrite_applied'))
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
      const result = await llmApi.chat(chatInput, selectedModel, promptValue)
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
    return <div>{t('editor_asset_not_found')}</div>
  }

  const category = asset?.category || 'content'
  if (category !== 'content') {
    return (
      <Card title={asset?.name || id}>
        <Space direction="vertical">
          <Tag color="blue">{category}</Tag>
          <p>{t('editor_not_markdown_type')}</p>
          <p>{t('editor_eval_hint')}</p>
          <p>{t('editor_metric_hint')}</p>
          <Button onClick={() => navigate(`/assets/${id}`)}>{t('editor_go_to_detail')}</Button>
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
            {updatedAt && <Tag color="blue" className="editor-time-tag">{t('editor_saved_at')} {formatUpdatedAt(updatedAt)}</Tag>}
          </div>
          <div className="editor-header-nav">
            <Button
              size="small"
              onClick={() => {
                if (hasChanges && !window.confirm(t('editor_unsaved_changes_warning'))) return
                navigate(`/assets/${id}/versions`)
              }}
            >
              {t('editor_v2_version_tree')}
            </Button>
            <Button
              size="small"
              onClick={() => {
                if (hasChanges && !window.confirm(t('editor_unsaved_changes_warning'))) return
                navigate(`/assets/${id}/eval`)
              }}
            >
              {t('editor_v2_run_eval')}
            </Button>
            <Button
              size="small"
              icon={<SwapOutlined />}
              onClick={() => {
                if (hasChanges && !window.confirm(t('editor_unsaved_changes_warning'))) return
                navigate('/compare')
              }}
            >
              {t('editor_v2_compare')}
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
              { key: 'editor', label: t('editor_tab_editor') },
              { key: 'diff', label: <span><DiffOutlined /> {t('editor_tab_diff')}</span> },
              { key: 'preview', label: t('editor_tab_preview') },
            ]}
          />
          <div className="toolbar-actions">
            {showRewriteInput ? (
              <>
                <Input
                  placeholder={t('editor_rewrite_placeholder')}
                  value={rewriteInstruction}
                  onChange={(e) => setRewriteInstruction(e.target.value)}
                  onPressEnter={handleRewrite}
                  style={{ width: 160 }}
                  size="small"
                  disabled={rewriting}
                />
                <Button size="small" type="primary" onClick={handleRewrite} loading={rewriting}>{t('editor_rewrite_apply')}</Button>
                <Button size="small" onClick={() => { setShowRewriteInput(false); setRewriteInstruction('') }} disabled={rewriting}>{t('common_cancel')}</Button>
              </>
            ) : (
              <>
                <Button icon={<PlayCircleOutlined />} onClick={handleValidate} loading={validating} size="small">
                  {t('editor_validate_button')}
                </Button>
                <Button size="small" icon={<EditOutlined />} onClick={() => setShowRewriteInput(true)}>
                  {t('editor_rewrite_button')}
                </Button>
                {hasChanges && (
                  <Button size="small" onClick={handleRestore}>
                    {t('editor_restore_button')}
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
                  {t('editor_save')}
                </Button>
                <Button icon={<SaveOutlined />} onClick={handleCommit} loading={saving} size="small">
                  {t('editor_commit_button')}
                </Button>
              </>
            )}
          </div>
        </div>

        {/* Monaco Editor */}
        <div className="editor-content">
          {activeTab === 'editor' && (
            <div style={{ height: '100%', position: 'relative' }}>
              {!monacoReady && (
                <div style={{ position: 'absolute', inset: 0, display: 'flex', justifyContent: 'center', alignItems: 'center', background: '#1e1e1e', zIndex: 10 }}>
                  <Spin size="large" indicator={<LoadingOutlined spin style={{ fontSize: 24, color: '#fff' }} />} />
                </div>
              )}
              <MonacoEditor
                height="100%"
                language="markdown"
                value={promptValue}
                onChange={(value) => setPromptValue(value || '')}
                theme="vs-dark"
                options={{ minimap: { enabled: false } }}
                onMount={() => setMonacoReady(true)}
              />
            </div>
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
                <div className="no-changes">{t('editor_no_changes')}</div>
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
          <span className="chat-title">{t('editor_v2_ai_assistant')}</span>
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
              placeholder={t('editor_v2_select_provider')}
            />
            <Button icon={<ClearOutlined />} size="small" onClick={handleClearChat} />
          </div>
        </div>

        {/* Chat Messages */}
        <div className="chat-messages">
          {chatMessages.length === 0 && (
            <div className="chat-empty">
              <p>{t('editor_v2_chat_empty')}</p>
              <p className="chat-empty-hint">{t('editor_v2_chat_empty_hint')}</p>
            </div>
          )}
          {chatMessages.map((msg) => {
            const { think, clean } = extractThinkContent(msg.content)
            return (
              <div key={msg.id} className={`chat-message ${msg.role}`}>
                <div className="message-bubble">
                  {think && (
                    <div className="think-block">
                      <button className="think-toggle" onClick={() => toggleThink(msg.id)}>
                        💭 {t('editor_v2_thinking')} {expandedThinks.has(msg.id) ? '▲' : '▼'}
                      </button>
                      {expandedThinks.has(msg.id) && (
                        <pre className="think-content">{think}</pre>
                      )}
                    </div>
                  )}
                  {clean && <ReactMarkdown remarkPlugins={[remarkGfm]}>{clean}</ReactMarkdown>}
                </div>
              </div>
            )
          })}
          {chatLoading && (
            <div className="chat-message assistant">
              <div className="message-bubble loading">
                <Spin size="small" /> {t('editor_v2_thinking')}
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
            placeholder={t('editor_v2_chat_input_placeholder')}
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
            <span className="inject-title">{t('editor_v2_variables_title')}</span>
            <Button size="small" icon={<PlusOutlined />} onClick={addVariable}>{t('editor_v2_add_variable')}</Button>
          </div>
          <div className="inject-variables">
            {variables.map((v, index) => (
              <div key={index} className="variable-row">
                <Input
                  placeholder={t('editor_v2_key_placeholder')}
                  value={v.key}
                  onChange={(e) => updateVariable(index, 'key', e.target.value)}
                  className="variable-key"
                />
                <span className="variable-separator">:</span>
                <Input
                  placeholder={t('editor_v2_value_placeholder')}
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
            {t('editor_inject_button')}
          </Button>
        </div>
      </div>

      {/* Conflict Modal */}
      <Modal
        title={t('editor_conflict_title')}
        open={!!conflictDraft}
        width={900}
        footer={null}
        onCancel={() => setConflictDraft(null)}
      >
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <p>{t('editor_conflict_description')}</p>
          <DiffEditor
            height={300}
            language="markdown"
            original={conflictServerContent || t('editor_server_version')}
            modified={conflictDraft?.content || ''}
            theme="vs-dark"
            options={{ minimap: { enabled: false }, renderSideBySide: true }}
          />
          <Space>
            <Button type="primary" onClick={() => handleSave(conflictDraft?.content)}>
              {t('editor_keep_local')}
            </Button>
            <Button onClick={handleAcceptServer}>
              {t('editor_accept_server')}
            </Button>
            <Button onClick={handleKeepLocal}>
              {t('editor_review_later')}
            </Button>
          </Space>
        </Space>
      </Modal>

      {/* Quick Eval Modal */}
      <QuickEvalModal
        assetId={id || ''}
        assetName={asset?.name || id || ''}
        open={showQuickEval}
        onClose={() => setShowQuickEval(false)}
      />

      {/* Rewrite Preview Modal */}
      <Modal
        title={t('editor_v2_rewrite_preview_title')}
        open={showRewritePreview}
        width={900}
        onCancel={handleCancelRewrite}
        footer={[
          <Button key="cancel" onClick={handleCancelRewrite}>
            {t('common_cancel')}
          </Button>,
          <Button key="apply" type="primary" onClick={handleApplyRewrite}>
            {t('editor_v2_apply_rewrite')}
          </Button>,
        ]}
      >
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <p>{t('editor_v2_review_rewritten')}</p>
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
