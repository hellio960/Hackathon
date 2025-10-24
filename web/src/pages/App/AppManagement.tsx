import { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  message,
} from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { appApi } from '@/api/app'
import type { AllowApp } from '@/api/types'
import dayjs from 'dayjs'

const { Option } = Select
const { TextArea } = Input

export default function AppManagement() {
  const [loading, setLoading] = useState(false)
  const [dataSource, setDataSource] = useState<AllowApp[]>([])
  const [modalVisible, setModalVisible] = useState(false)
  const [editingApp, setEditingApp] = useState<AllowApp | null>(null)
  const [form] = Form.useForm()

  const columns: ColumnsType<AllowApp> = [
    {
      title: '应用名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '节点类型',
      dataIndex: 'nodeType',
      key: 'nodeType',
    },
    {
      title: 'Kodo路径',
      dataIndex: 'path',
      key: 'path',
    },
    {
      title: '描述',
      dataIndex: 'desc',
      key: 'desc',
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
        <Space>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ]

  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await appApi.getAllowAppsList()
      setDataSource(res.apps || [])
    } catch (error) {
      console.error('获取应用列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleAdd = () => {
    setEditingApp(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (app: AllowApp) => {
    setEditingApp(app)
    form.setFieldsValue({
      name: app.name,
      nodeTypes: [app.nodeType],
      path: app.path,
      desc: app.desc,
    })
    setModalVisible(true)
  }

  const handleDelete = (app: AllowApp) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除应用 "${app.name}" 吗？`,
      okType: 'danger',
      onOk: async () => {
        try {
          await appApi.updateAllowApps({
            operation: 'del',
            id: app.id,
          })
          message.success('删除成功')
          fetchData()
        } catch (error) {
          console.error('删除失败:', error)
        }
      },
    })
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      const operation = editingApp ? 'update' : 'add'

      await appApi.updateAllowApps({
        operation,
        id: editingApp?.id,
        name: values.name,
        nodeTypes: values.nodeTypes,
        path: values.path,
        desc: values.desc,
      })

      message.success(editingApp ? '更新成功' : '添加成功')
      setModalVisible(false)
      fetchData()
    } catch (error) {
      console.error('提交失败:', error)
    }
  }

  return (
    <div>
      <Card
        title="应用管理"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            添加应用
          </Button>
        }
      >
        <Table
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          rowKey="id"
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
        />
      </Card>

      <Modal
        title={editingApp ? '编辑应用' : '添加应用'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="应用名称"
            rules={[{ required: true, message: '请输入应用名称' }]}
          >
            <Input placeholder="例如: example-app" disabled={!!editingApp} />
          </Form.Item>

          <Form.Item
            name="nodeTypes"
            label="节点类型"
            rules={[{ required: true, message: '请选择节点类型' }]}
          >
            <Select mode="multiple" placeholder="选择节点类型">
              <Option value="node">大节点</Option>
              <Option value="smallBox">小盒子</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="path"
            label="Kodo存储路径"
            rules={[{ required: true, message: '请输入Kodo路径' }]}
          >
            <Input placeholder="例如: apps/example-app/" />
          </Form.Item>

          <Form.Item name="desc" label="描述">
            <TextArea rows={3} placeholder="应用描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
