import { useState } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu } from 'antd'
import {
  RocketOutlined,
  AppstoreOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import type { MenuProps } from 'antd'

const { Header, Content, Sider } = Layout

type MenuItem = Required<MenuProps>['items'][number]

const items: MenuItem[] = [
  {
    key: '/releases',
    icon: <RocketOutlined />,
    label: '发布管理',
  },
  {
    key: '/apps',
    icon: <AppstoreOutlined />,
    label: '应用管理',
  },
]

export default function AppLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(false)

  const handleMenuClick: MenuProps['onClick'] = (e) => {
    navigate(e.key)
  }

  const selectedKey = '/' + location.pathname.split('/')[1]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible collapsed={collapsed} onCollapse={setCollapsed}>
        <div
          style={{
            height: 32,
            margin: 16,
            color: '#fff',
            fontSize: 18,
            fontWeight: 'bold',
            textAlign: 'center',
          }}
        >
          {collapsed ? 'J' : 'Jarvis'}
        </div>
        <Menu
          theme="dark"
          selectedKeys={[selectedKey]}
          mode="inline"
          items={items}
          onClick={handleMenuClick}
        />
      </Sider>
      <Layout>
        <Header style={{ padding: '0 24px', background: '#fff' }}>
          <h1 style={{ margin: 0, fontSize: 20 }}>Jarvis 智能发布系统</h1>
        </Header>
        <Content style={{ margin: '16px' }}>
          <div style={{ padding: 24, minHeight: 360, background: '#fff' }}>
            <Outlet />
          </div>
        </Content>
      </Layout>
    </Layout>
  )
}
