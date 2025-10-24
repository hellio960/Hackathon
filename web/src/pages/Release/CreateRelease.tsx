import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Form,
  Input,
  Select,
  InputNumber,
  Button,
  Space,
  message,
  Radio,
  Divider,
} from 'antd'
import { releaseApi } from '@/api/release'
import type { NodeReleaseCreateReq, PackageInfo } from '@/api/types'

const { Option } = Select
const { TextArea } = Input

export default function CreateRelease() {
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [packages, setPackages] = useState<PackageInfo[]>([])
  const [filterMode, setFilterMode] = useState<'nodeIds' | 'filter'>('filter')

  const fetchPackages = async (devType: string, app: string) => {
    try {
      const res = await releaseApi.getPackages({
        devType,
        app,
        size: 10,
      })
      setPackages(res.packages || [])
    } catch (error) {
      console.error('获取包列表失败:', error)
    }
  }

  const handleDevTypeChange = (devType: string) => {
    const app = form.getFieldValue('appName')
    if (app) {
      fetchPackages(devType, app)
    }
  }

  const handleAppNameChange = (app: string) => {
    const devType = form.getFieldValue('devType')
    if (devType) {
      fetchPackages(devType, app)
    }
  }

  const handlePackageSelect = (url: string) => {
    const pkg = packages.find((p) => p.url === url)
    if (pkg) {
      form.setFieldsValue({
        appConfig: {
          url: pkg.url,
          md5: pkg.md5,
        },
      })
    }
  }

  const handleSubmit = async (values: any) => {
    setLoading(true)
    try {
      const data: NodeReleaseCreateReq = {
        devType: values.devType,
        appName: values.appName,
        describe: values.describe,
        releaseType: values.releaseType,
        opType: values.opType,
        appConfig: {
          url: values.appConfig.url,
          type: values.appConfig.type,
          cmd: values.appConfig.cmd,
          args: values.appConfig.args
            ? values.appConfig.args.split(',').map((s: string) => s.trim())
            : [],
          dir: values.appConfig.dir,
          healthUrl: values.appConfig.healthUrl,
          md5: values.appConfig.md5,
        },
        grayPolicy: {
          percentage: values.grayPolicy.percentage,
          ...(filterMode === 'filter'
            ? {
                filter: {
                  status: values.grayPolicy.filter.status,
                  stages: values.grayPolicy.filter.stages || [],
                },
              }
            : {
                nodeIds: values.grayPolicy.nodeIds
                  ? values.grayPolicy.nodeIds.split(',').map((s: string) => s.trim())
                  : [],
              }),
        },
      }

      await releaseApi.createRelease(data)
      message.success('创建发布任务成功')
      navigate('/releases')
    } catch (error) {
      console.error('创建发布任务失败:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card
      title="创建发布任务"
      extra={
        <Button onClick={() => navigate('/releases')}>返回列表</Button>
      }
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        initialValues={{
          releaseType: 'formal',
          opType: 'update',
          appConfig: {
            type: 'tar.gz',
          },
          grayPolicy: {
            percentage: 10,
            filter: {
              status: 'online',
            },
          },
        }}
      >
        <Divider>基本信息</Divider>
        <Form.Item
          name="devType"
          label="设备类型"
          rules={[{ required: true, message: '请选择设备类型' }]}
        >
          <Input
            placeholder="例如: node, k8s-node"
            onChange={(e) => handleDevTypeChange(e.target.value)}
          />
        </Form.Item>

        <Form.Item
          name="appName"
          label="应用名称"
          rules={[{ required: true, message: '请输入应用名称' }]}
        >
          <Input
            placeholder="例如: example-app"
            onChange={(e) => handleAppNameChange(e.target.value)}
          />
        </Form.Item>

        <Form.Item
          name="describe"
          label="发布说明"
          rules={[{ required: true, message: '请输入发布说明' }]}
        >
          <TextArea rows={3} placeholder="描述本次发布的内容" />
        </Form.Item>

        <Space size="large">
          <Form.Item
            name="releaseType"
            label="发布类型"
            rules={[{ required: true }]}
          >
            <Radio.Group>
              <Radio value="formal">正式发布</Radio>
              <Radio value="beta">功能验证</Radio>
            </Radio.Group>
          </Form.Item>

          <Form.Item
            name="opType"
            label="操作类型"
            rules={[{ required: true }]}
          >
            <Radio.Group>
              <Radio value="add">新增组件</Radio>
              <Radio value="update">升级组件</Radio>
              <Radio value="delete">移除组件</Radio>
            </Radio.Group>
          </Form.Item>
        </Space>

        <Divider>应用配置</Divider>

        {packages.length > 0 && (
          <Form.Item label="选择包">
            <Select
              placeholder="选择一个包"
              onChange={handlePackageSelect}
              showSearch
              optionFilterProp="children"
            >
              {packages.map((pkg) => (
                <Option key={pkg.url} value={pkg.url}>
                  {pkg.file} ({pkg.size})
                </Option>
              ))}
            </Select>
          </Form.Item>
        )}

        <Form.Item
          name={['appConfig', 'url']}
          label="包地址"
          rules={[{ required: true, message: '请输入包地址' }]}
        >
          <Input placeholder="https://kodo.example.com/packages/app-v1.0.0.tar.gz" />
        </Form.Item>

        <Form.Item
          name={['appConfig', 'type']}
          label="包类型"
          rules={[{ required: true }]}
        >
          <Select>
            <Option value="tar.gz">tar.gz</Option>
            <Option value="zip">zip</Option>
            <Option value="executable">executable</Option>
          </Select>
        </Form.Item>

        <Form.Item
          name={['appConfig', 'cmd']}
          label="启动命令"
          rules={[{ required: true, message: '请输入启动命令' }]}
        >
          <Input placeholder="例如: example-app" />
        </Form.Item>

        <Form.Item name={['appConfig', 'args']} label="启动参数">
          <Input placeholder="多个参数用逗号分隔，例如: --config,/etc/app/config.yaml" />
        </Form.Item>

        <Form.Item
          name={['appConfig', 'dir']}
          label="工作目录"
          rules={[{ required: true, message: '请输入工作目录' }]}
        >
          <Input placeholder="例如: /opt/app" />
        </Form.Item>

        <Form.Item name={['appConfig', 'healthUrl']} label="健康检查URL">
          <Input placeholder="例如: /health" />
        </Form.Item>

        <Form.Item
          name={['appConfig', 'md5']}
          label="MD5校验"
          rules={[{ required: true, message: '请输入MD5' }]}
        >
          <Input placeholder="包的MD5值" />
        </Form.Item>

        <Divider>灰度策略</Divider>

        <Form.Item label="灰度模式">
          <Radio.Group
            value={filterMode}
            onChange={(e) => setFilterMode(e.target.value)}
          >
            <Radio value="filter">规则过滤</Radio>
            <Radio value="nodeIds">指定节点</Radio>
          </Radio.Group>
        </Form.Item>

        {filterMode === 'filter' ? (
          <>
            <Form.Item
              name={['grayPolicy', 'filter', 'status']}
              label="节点状态"
              rules={[{ required: true }]}
            >
              <Radio.Group>
                <Radio value="online">在线</Radio>
                <Radio value="outline">离线</Radio>
              </Radio.Group>
            </Form.Item>

            <Form.Item
              name={['grayPolicy', 'filter', 'stages']}
              label="节点阶段"
            >
              <Select mode="tags" placeholder="输入阶段标识，例如: prod, test">
                <Option value="prod">prod</Option>
                <Option value="test">test</Option>
                <Option value="dev">dev</Option>
              </Select>
            </Form.Item>
          </>
        ) : (
          <Form.Item
            name={['grayPolicy', 'nodeIds']}
            label="节点ID列表"
            rules={[{ required: true, message: '请输入节点ID' }]}
          >
            <TextArea
              rows={4}
              placeholder="多个节点ID用逗号分隔，例如: node-001, node-002"
            />
          </Form.Item>
        )}

        <Form.Item
          name={['grayPolicy', 'percentage']}
          label="灰度比例"
          rules={[{ required: true }]}
        >
          <InputNumber
            min={0}
            max={100}
            formatter={(value) => `${value}%`}
            style={{ width: 200 }}
          />
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" loading={loading}>
              创建发布
            </Button>
            <Button onClick={() => navigate('/releases')}>取消</Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  )
}
