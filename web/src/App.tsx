import { useEffect, useRef } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from 'antd'
import Sidebar from './components/Sidebar'
import AssetListView from './views/AssetListView'
import EditorViewV2 from './views/EditorViewV2'
import CreateAssetView from './views/CreateAssetView'
import VersionTreeView from './views/VersionTreeView'
import EvalLayout from './views/eval/EvalLayout'
import EvalDesignView from './views/eval/EvalDesignView'
import EvalRunView from './views/eval/EvalRunView'
import EvalReportView from './views/eval/EvalReportView'
import EvalHistoryView from './views/eval/EvalHistoryView'
import CompareView from './views/CompareView'
import SettingsView from './views/SettingsView'
import AssetDetailRouter from './views/AssetDetailRouter'
import ExecutionListView from './views/ExecutionListView'
import CallLogView from './views/CallLogView'
import { loadAssetTypesFromAPI } from './config/assetTypes'
import { loadTagsFromAPI } from './config/tags'
import { loadLLMConfigsFromAPI } from './config/llmConfig'
import { useExecutionPolling } from './views/eval/hooks/useExecutionPolling'

const { Content } = Layout

function App() {
  const initRef = useRef(false)

  // Global polling for all running eval executions
  useExecutionPolling()

  useEffect(() => {
    if (initRef.current) return
    initRef.current = true
    loadAssetTypesFromAPI()
    loadTagsFromAPI()
    loadLLMConfigsFromAPI()
  }, [])

  return (
    <Layout style={{ height: '100vh' }}>
      <Sidebar />
      <Layout style={{ padding: '0', height: 'calc(100vh - 56px)', display: 'flex', flexDirection: 'column', overflow: 'hidden', boxSizing: 'border-box' }}>
        <Content style={{ height: '100%', overflow: 'auto', padding: 8 }}>
          <Routes>
            <Route path="/" element={<Navigate to="/assets" replace />} />
            <Route path="/assets" element={<AssetListView />} />
            <Route path="/assets/new" element={<CreateAssetView />} />
            <Route path="/assets/:id" element={<AssetDetailRouter />} />
            <Route path="/assets/:id/edit" element={<EditorViewV2 />} />
            <Route path="/assets/:id/versions" element={<VersionTreeView />} />
            <Route path="/assets/:id/eval" element={<EvalLayout />}>
              <Route index element={<EvalRunView />} />
              <Route path="design" element={<EvalDesignView />} />
              <Route path="run" element={<EvalRunView />} />
              <Route path="report/:runId" element={<EvalReportView />} />
              <Route path="history" element={<EvalHistoryView />} />
            </Route>
            <Route path="/executions" element={<ExecutionListView />} />
            <Route path="/executions/:id/calls" element={<CallLogView />} />
            <Route path="/compare" element={<CompareView />} />
            <Route path="/settings" element={<SettingsView />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  )
}

export default App
