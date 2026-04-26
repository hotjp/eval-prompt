import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Input, Button, Space, message, Spin, Tabs, Tag, Modal } from 'antd'
import { SaveOutlined, PlayCircleOutlined, SwapOutlined, DiffOutlined, EditOutlined } from '@ant-design/icons'
import MonacoEditor, { DiffEditor } from '@monaco-editor/react'
import { assetApi, triggerApi, llmApi } from '../api/client'
import type { AssetDetail } from '../api/client'

const { TextArea } = Input

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

// Default body template — frontmatter is handled server-side
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

function getDraftKey(id: string) {
  return `draft:${id}`
}

function loadDraft(id: string): Draft | null {
  try {
    const raw = localStorage.getItem(getDraftKey(id))
    if (!raw) return null
    const draft: Draft = JSON.parse(raw)
    // Expire after 7 days
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

function EditorView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [promptValue, setPromptValue] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [hasChanges, setHasChanges] = useState(false)
  const [variablesValue, setVariablesValue] = useState('{}\n\n{\n  "key": "value"\n}')
  const [injectedResult, setInjectedResult] = useState('')
  const [validating, setValidating] = useState(false)
  const [validationResult, setValidationResult] = useState<{ valid: boolean; message?: string } | null>(null)
  const [contentHash, setContentHash] = useState('')
  const [updatedAt, setUpdatedAt] = useState('')
  const [conflictDraft, setConflictDraft] = useState<Draft | null>(null)
  const [conflictServerContent, setConflictServerContent] = useState('')
  const [showRewriteInput, setShowRewriteInput] = useState(false)
  const [rewriteInstruction, setRewriteInstruction] = useState('')
  const [rewriting, setRewriting] = useState(false)
  const loadedRef = useRef(false)
  const autoSaveTimer = useRef<number | undefined>(undefined)

  // Auto-save draft to localStorage on content change
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

    // Prevent double-loading
    if (loadedRef.current) {
      return
    }

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

        // Check for local draft
        const draft = loadDraft(id)
        const serverContent = contentData.content || defaultBody
        const serverHash = contentData.content_hash || ''

        if (draft && draft.savedHash && draft.savedHash !== serverHash) {
          // Draft exists and server has changed — show conflict
          setConflictDraft(draft)
          setConflictServerContent(serverContent)
          setPromptValue(draft.content)
          setOriginalContent(draft.content)
        } else {
          // No conflict — use server content (or draft if server has no content)
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
      // Use content from Preference-Applied response directly
      setPromptValue(result.content)
      setOriginalContent(result.content)
      setContentHash(result.content_hash)
      setUpdatedAt(result.updated_at)
      setHasChanges(false)
      clearDraft(id)
    } catch (err: any) {
      if (err?.response?.status === 409) {
        message.warning('Conflict detected. Please choose which version to keep.')
        // Reload server content and show conflict
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
      setValidationResult(result)
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
      const vars = JSON.parse(variablesValue.replace(/^\{/, '').replace(/\}$/, ''))
      const result = await triggerApi.inject(promptValue, vars)
      setInjectedResult(result.result)
    } catch {
      message.error('Failed to inject variables')
    }
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
      const result = await llmApi.rewrite(promptValue, rewriteInstruction)
      setPromptValue(result.rewritten)
      setShowRewriteInput(false)
      setRewriteInstruction('')
      message.success('Rewrite applied')
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

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  if (!asset && id !== 'new') {
    return <div>Asset not found</div>
  }

  // For eval/metric assets, show a message that editing is done via dedicated views
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
    <div>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Card
          title={asset?.name || id}
          extra={
            <Space>
              {asset?.asset_type && <Tag>{asset.asset_type}</Tag>}
              {asset?.state && <Tag color={asset.state === 'active' ? 'green' : 'orange'}>{asset.state}</Tag>}
              {updatedAt && <Tag color="blue">Saved {formatUpdatedAt(updatedAt)}</Tag>}
              <Button icon={<SaveOutlined />} onClick={handleCommit} loading={saving}>
                Commit
              </Button>
            </Space>
          }
        >
          <p>{asset?.description || 'New prompt asset'}</p>
        </Card>

        <Tabs
          defaultActiveKey="editor"
          items={[
            {
              key: 'editor',
              label: 'Editor',
              children: (
                <Card>
                  <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                    <MonacoEditor
                      height="400px"
                      language="markdown"
                      value={promptValue}
                      onChange={(value) => setPromptValue(value || '')}
                      theme="vs-dark"
                      options={{ minimap: { enabled: false } }}
                    />
                    <Space>
                      <Button
                        type="primary"
                        icon={<SaveOutlined />}
                        onClick={() => handleSave()}
                        loading={saving}
                        disabled={!hasChanges}
                      >
                        Save
                      </Button>
                      <Button
                        icon={<PlayCircleOutlined />}
                        onClick={handleValidate}
                        loading={validating}
                      >
                        Validate
                      </Button>
                      <Button
                        icon={<EditOutlined />}
                        onClick={() => setShowRewriteInput(!showRewriteInput)}
                      >
                        Rewrite
                      </Button>
                      {hasChanges && (
                        <Button onClick={handleRestore}>
                          Restore
                        </Button>
                      )}
                    </Space>
                    {showRewriteInput && (
                      <Space>
                        <Input
                          placeholder="Enter rewrite instruction (e.g., 改成更礼貌, 缩短, 扩充)"
                          value={rewriteInstruction}
                          onChange={(e) => setRewriteInstruction(e.target.value)}
                          onPressEnter={handleRewrite}
                          style={{ width: 300 }}
                          disabled={rewriting}
                        />
                        <Button type="primary" onClick={handleRewrite} loading={rewriting}>
                          Apply
                        </Button>
                        <Button onClick={() => { setShowRewriteInput(false); setRewriteInstruction('') }} disabled={rewriting}>
                          Cancel
                        </Button>
                      </Space>
                    )}
                    {validationResult && (
                      <Tag color={validationResult.valid ? 'green' : 'red'}>
                        {validationResult.valid ? 'Valid' : validationResult.message}
                      </Tag>
                    )}
                  </Space>
                </Card>
              ),
            },
            {
              key: 'diff',
              label: (
                <span>
                  <DiffOutlined /> Diff
                </span>
              ),
              children: (
                <Card title="Changes">
                  {hasChanges ? (
                    <DiffEditor
                      height="400px"
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
                    <div style={{ color: '#888', padding: 16 }}>No changes to show</div>
                  )}
                </Card>
              ),
            },
            {
              key: 'preview',
              label: 'Preview',
              children: (
                <Card title="Variable Injection">
                  <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                    <TextArea
                      rows={4}
                      value={variablesValue}
                      onChange={(e) => setVariablesValue(e.target.value)}
                      placeholder="Enter variables as JSON"
                    />
                    <Button type="primary" onClick={handleInject}>
                      Inject
                    </Button>
                    {injectedResult && (
                      <Card title="Result" style={{ background: '#f5f5f5' }}>
                        <pre>{injectedResult}</pre>
                      </Card>
                    )}
                  </Space>
                </Card>
              ),
            },
          ]}
        />

        <Card title="Snapshots">
          <Space>
            <Button onClick={() => navigate(`/assets/${id}/versions`)}>
              View Version Tree
            </Button>
            <Button onClick={() => navigate(`/assets/${id}/eval`)}>
              Run Evaluation
            </Button>
            <Button icon={<SwapOutlined />} onClick={() => navigate('/compare')}>
              Compare
            </Button>
          </Space>
        </Card>
      </Space>

      {/* Conflict resolution modal */}
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
    </div>
  )
}

export default EditorView
