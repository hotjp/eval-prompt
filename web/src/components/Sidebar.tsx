import { Layout, Menu } from 'antd'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  AppstoreOutlined,
  EditOutlined,
  HistoryOutlined,
  CheckCircleOutlined,
  SwapOutlined,
} from '@ant-design/icons'

const { Sider } = Layout

const menuItems = [
  { key: '/assets', icon: <AppstoreOutlined />, label: 'Assets' },
  { key: '/assets/:id/edit', icon: <EditOutlined />, label: 'Editor' },
  { key: '/assets/:id/versions', icon: <HistoryOutlined />, label: 'Versions' },
  { key: '/assets/:id/eval', icon: <CheckCircleOutlined />, label: 'Evaluation' },
  { key: '/compare', icon: <SwapOutlined />, label: 'Compare' },
]

function Sidebar() {
  const navigate = useNavigate()
  const location = useLocation()

  const getSelectedKey = () => {
    if (location.pathname.startsWith('/assets/') && location.pathname.endsWith('/edit')) {
      return '/assets/:id/edit'
    }
    if (location.pathname.startsWith('/assets/') && location.pathname.endsWith('/versions')) {
      return '/assets/:id/versions'
    }
    if (location.pathname.startsWith('/assets/') && location.pathname.endsWith('/eval')) {
      return '/assets/:id/eval'
    }
    return location.pathname
  }

  const handleMenuClick = ({ key }: { key: string }) => {
    if (key.startsWith('/assets/:id')) {
      const assetId = location.pathname.split('/')[2]
      if (assetId) {
        navigate(key.replace(':id', assetId))
      }
    } else {
      navigate(key)
    }
  }

  return (
    <Sider width={200} style={{ background: '#fff' }}>
      <Menu
        mode="inline"
        selectedKeys={[getSelectedKey()]}
        items={menuItems}
        onClick={handleMenuClick}
        style={{ height: '100%', borderRight: 0 }}
      />
    </Sider>
  )
}

export default Sidebar
