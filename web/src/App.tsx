import { useEffect, useRef } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from 'antd'
import Sidebar from './components/Sidebar'
import AssetListView from './views/AssetListView'
import EditorView from './views/EditorView'
import CreateAssetView from './views/CreateAssetView'
import VersionTreeView from './views/VersionTreeView'
import EvalPanelView from './views/EvalPanelView'
import CompareView from './views/CompareView'
import SettingsView from './views/SettingsView'
import AssetDetailRouter from './views/AssetDetailRouter'
import ExecutionListView from './views/ExecutionListView'
import CallLogView from './views/CallLogView'
import { loadAssetTypesFromAPI } from './config/bizLines'
import { loadTagsFromAPI } from './config/tags'

const { Content } = Layout

function App() {
  const initRef = useRef(false)

  useEffect(() => {
    if (initRef.current) return
    initRef.current = true
    loadAssetTypesFromAPI()
    loadTagsFromAPI()
  }, [])

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sidebar />
      <Layout style={{ padding: '0' }}>
        <Content style={{ padding: '24px' }}>
          <Routes>
            <Route path="/" element={<Navigate to="/assets" replace />} />
            <Route path="/assets" element={<AssetListView />} />
            <Route path="/assets/new" element={<CreateAssetView />} />
            <Route path="/assets/:id" element={<AssetDetailRouter />} />
            <Route path="/assets/:id/edit" element={<EditorView />} />
            <Route path="/assets/:id/versions" element={<VersionTreeView />} />
            <Route path="/assets/:id/eval" element={<EvalPanelView />} />
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
