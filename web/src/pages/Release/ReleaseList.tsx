import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Table,
  Button,
  Space,
  Tag,
  Input,
  Select,
  Card,
  message,
} from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { releaseApi } from '@/api/release'
import type { NodeReleaseBrief } from '@/api/types'
import dayjs from 'dayjs'

const { Search } = Input
const { Option } = Select

const stateColorMap: Record<string, string> = {
  processing: 'blue',
  completed: 'green',
  rollbacked: 'red',
  paused: 'orange',
}

const stateTextMap: Record<string, string> = {
  processing: '进行中',
  completed: '已完成',
  rollbacked: '已回滚',
  paused: '已暂停',
}

export default function ReleaseList() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [dataSource, setDataSource] = useState<NodeReleaseBrief[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [app, setApp] = useState<string>()
  const [states, setStates] = useState<string[]>()

  const columns: ColumnsType<NodeReleaseBrief> = [
    {
      title: '任务ID',
      dataIndex: 'id',
      key: 'id',
      width: 100,
      render: (id: string) => (
        <Button type="link" onClick={() => navigate(`/releases/${id}`)}>
          {id.substring(0, 8)}
        </Button>
      ),
    },
    {
      title: '应用名称',
      dataIndex: 'app',
      key: 'app',
    },
    {
      title: '设备类型',
      dataIndex: 'deviceType',
      key: 'deviceType',
    },
    {
      title: '发布类型',
      dataIndex: 'releaseType',
      key: 'releaseType',
      render: (type: string) => (
        <Tag color={type === 'formal' ? 'blue' : 'orange'}>
          {type === 'formal' ? '正式发布' : '功能验证'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'state',
      key: 'state',
      render: (state: string) => (
        <Tag color={stateColorMap[state] || 'default'}>
          {stateTextMap[state] || state}
        </Tag>
      ),
    },
    {
      title: '操作人',
      dataIndex: 'operator',
      key: 'operator',
    },
    {
      title: '创建时间',
      dataIndex: 'createAt',
      key: 'createAt',
      render: (time: number) => dayjs(time * 1000).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Button type="link" onClick={() => navigate(`/releases/${record.id}`)}>
          查看详情
        </Button>
      ),
    },
  ]

  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await releaseApi.getReleaseList({
        app,
        states,
        page,
        size: pageSize,
      })
      setDataSource(res.items || [])
      setTotal(res.total || 0)
    } catch (error) {
      console.error('获取发布列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, pageSize, app, states])

  return (
    <div>
      <Card
        title="发布任务列表"
        extra={
          <Space>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => navigate('/releases/create')}
            >
              创建发布
            </Button>
            <Button icon={<ReloadOutlined />} onClick={fetchData}>
              刷新
            </Button>
          </Space>
        }
      >
        <Space style={{ marginBottom: 16 }}>
          <Search
            placeholder="搜索应用名称"
            allowClear
            style={{ width: 200 }}
            onSearch={setApp}
          />
          <Select
            mode="multiple"
            placeholder="选择状态"
            style={{ width: 200 }}
            allowClear
            onChange={setStates}
          >
            <Option value="processing">进行中</Option>
            <Option value="completed">已完成</Option>
            <Option value="rollbacked">已回滚</Option>
            <Option value="paused">已暂停</Option>
          </Select>
        </Space>

        <Table
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          rowKey="id"
          pagination={{
            current: page,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) => {
              setPage(page)
              setPageSize(pageSize)
            },
          }}
        />
      </Card>
    </div>
  )
}
