import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Card,
  Descriptions,
  Tag,
  Button,
  Space,
  Modal,
  InputNumber,
  message,
  Tabs,
  Table,
  Progress,
} from 'antd'
import {
  RollbackOutlined,
  CheckOutlined,
  ArrowRightOutlined,
  HistoryOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { releaseApi } from '@/api/release'
import type { NodeReleaseDetail, NodeReleaseHistoryItem } from '@/api/types'
import dayjs from 'dayjs'

const { TabPane } = Tabs

export default function ReleaseDetail() {
  const { releaseId } = useParams<{ releaseId: string }>()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [detail, setDetail] = useState<NodeReleaseDetail>()
  const [history, setHistory] = useState<NodeReleaseHistoryItem[]>([])
  const [continueModalVisible, setContinueModalVisible] = useState(false)
  const [newPercentage, setNewPercentage] = useState(0)

  const fetchDetail = async () => {
    if (!releaseId) return
    setLoading(true)
    try {
      const res = await releaseApi.getReleaseDetail(releaseId)
      setDetail(res)
      setNewPercentage(res.grayPolicy.percentage)
    } catch (error) {
      console.error('获取发布详情失败:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchHistory = async () => {
    if (!releaseId) return
    try {
      const res = await releaseApi.getReleaseHistory(releaseId)
      setHistory(res.items || [])
    } catch (error) {
      console.error('获取发布历史失败:', error)
    }
  }

  useEffect(() => {
    fetchDetail()
    fetchHistory()
  }, [releaseId])

  const handleContinue = async () => {
    if (!releaseId) return
    try {
      await releaseApi.continueRelease(releaseId, {
        percentage: newPercentage,
      })
      message.success('继续发布成功')
      setContinueModalVisible(false)
      fetchDetail()
      fetchHistory()
    } catch (error) {
      console.error('继续发布失败:', error)
    }
  }

  const handleComplete = () => {
    if (!releaseId) return
    Modal.confirm({
      title: '确认全量发布',
      content: '全量发布后将无法继续灰度，确定要全量发布吗？',
      onOk: async () => {
        try {
          await releaseApi.completeRelease(releaseId)
          message.success('全量发布成功')
          fetchDetail()
          fetchHistory()
        } catch (error) {
          console.error('全量发布失败:', error)
        }
      },
    })
  }

  const handleRollback = () => {
    if (!releaseId) return
    Modal.confirm({
      title: '确认回滚',
      content: '回滚后将恢复到上一个版本，确定要回滚吗？',
      okType: 'danger',
      onOk: async () => {
        try {
          await releaseApi.rollbackRelease(releaseId)
          message.success('回滚成功')
          fetchDetail()
          fetchHistory()
        } catch (error) {
          console.error('回滚失败:', error)
        }
      },
    })
  }

  const historyColumns: ColumnsType<NodeReleaseHistoryItem> = [
    {
      title: '操作',
      dataIndex: 'operation',
      key: 'operation',
    },
    {
      title: '操作人',
      dataIndex: 'operator',
      key: 'operator',
    },
    {
      title: '操作时间',
      dataIndex: 'opTime',
      key: 'opTime',
      render: (time: number) => dayjs(time * 1000).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '状态变更',
      key: 'stateChange',
      render: (_, record) => (
        <span>
          {record.beforeState} <ArrowRightOutlined /> {record.afterState}
        </span>
      ),
    },
    {
      title: '备注',
      dataIndex: 'remark',
      key: 'remark',
    },
  ]

  if (!detail) {
    return <Card loading={loading}>加载中...</Card>
  }

  const progressPercent = detail.grayPolicy.percentage

  return (
    <div>
      <Card
        title={`发布任务详情 - ${detail.app}`}
        extra={
          <Space>
            <Button onClick={() => navigate('/releases')}>返回列表</Button>
            {detail.state === 'processing' && (
              <>
                <Button
                  type="primary"
                  icon={<ArrowRightOutlined />}
                  onClick={() => setContinueModalVisible(true)}
                >
                  继续发布
                </Button>
                <Button
                  type="primary"
                  icon={<CheckOutlined />}
                  onClick={handleComplete}
                >
                  全量发布
                </Button>
              </>
            )}
            {detail.rollbackAllowed && (
              <Button
                danger
                icon={<RollbackOutlined />}
                onClick={handleRollback}
              >
                回滚
              </Button>
            )}
          </Space>
        }
      >
        <Descriptions column={2} bordered>
          <Descriptions.Item label="任务ID">{detail.id}</Descriptions.Item>
          <Descriptions.Item label="应用名称">{detail.app}</Descriptions.Item>
          <Descriptions.Item label="设备类型">
            {detail.deviceType}
          </Descriptions.Item>
          <Descriptions.Item label="发布类型">
            <Tag color={detail.releaseType === 'formal' ? 'blue' : 'orange'}>
              {detail.releaseType === 'formal' ? '正式发布' : '功能验证'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="操作类型">
            {detail.opType === 'add' && '新增组件'}
            {detail.opType === 'update' && '升级组件'}
            {detail.opType === 'delete' && '移除组件'}
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag
              color={
                detail.state === 'processing'
                  ? 'blue'
                  : detail.state === 'completed'
                  ? 'green'
                  : 'red'
              }
            >
              {detail.state === 'processing' && '进行中'}
              {detail.state === 'completed' && '已完成'}
              {detail.state === 'rollbacked' && '已回滚'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="灰度进度" span={2}>
            <Progress percent={progressPercent} status="active" />
            <div style={{ marginTop: 8 }}>
              当前灰度节点数: {detail.allowNodesTotal} / 总节点数:{' '}
              {detail.grayPolicy.statInfo?.totalCount || 0}
            </div>
          </Descriptions.Item>
          <Descriptions.Item label="操作人">
            {detail.operator}
          </Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {dayjs(detail.createAt * 1000).format('YYYY-MM-DD HH:mm:ss')}
          </Descriptions.Item>
          <Descriptions.Item label="更新时间">
            {dayjs(detail.updateAt * 1000).format('YYYY-MM-DD HH:mm:ss')}
          </Descriptions.Item>
          <Descriptions.Item label="完成时间">
            {detail.endAt
              ? dayjs(detail.endAt * 1000).format('YYYY-MM-DD HH:mm:ss')
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="发布说明" span={2}>
            {detail.desc}
          </Descriptions.Item>
        </Descriptions>

        <Tabs defaultActiveKey="config" style={{ marginTop: 24 }}>
          <TabPane tab="配置信息" key="config">
            {detail.mainConfig && (
              <Card title="当前版本配置" style={{ marginBottom: 16 }}>
                <Descriptions column={1}>
                  <Descriptions.Item label="包地址">
                    {detail.mainConfig.url}
                  </Descriptions.Item>
                  <Descriptions.Item label="启动命令">
                    {detail.mainConfig.cmd}
                  </Descriptions.Item>
                  <Descriptions.Item label="工作目录">
                    {detail.mainConfig.dir}
                  </Descriptions.Item>
                  <Descriptions.Item label="MD5">
                    {detail.mainConfig.md5}
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            )}
            {detail.alterConfig && (
              <Card title="灰度版本配置">
                <Descriptions column={1}>
                  <Descriptions.Item label="包地址">
                    {detail.alterConfig.url}
                  </Descriptions.Item>
                  <Descriptions.Item label="启动命令">
                    {detail.alterConfig.cmd}
                  </Descriptions.Item>
                  <Descriptions.Item label="工作目录">
                    {detail.alterConfig.dir}
                  </Descriptions.Item>
                  <Descriptions.Item label="MD5">
                    {detail.alterConfig.md5}
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            )}
          </TabPane>
          <TabPane tab="灰度节点" key="nodes">
            <div>
              <p>当前灰度节点数: {detail.allowNodesTotal}</p>
              <div style={{ marginTop: 16 }}>
                {detail.allowNodes.slice(0, 100).map((node) => (
                  <Tag key={node} style={{ marginBottom: 8 }}>
                    {node}
                  </Tag>
                ))}
                {detail.allowNodesTotal > 100 && (
                  <div style={{ marginTop: 8 }}>
                    ...还有 {detail.allowNodesTotal - 100} 个节点
                  </div>
                )}
              </div>
            </div>
          </TabPane>
          <TabPane
            tab={
              <span>
                <HistoryOutlined /> 操作历史
              </span>
            }
            key="history"
          >
            <Table
              columns={historyColumns}
              dataSource={history}
              rowKey="opTime"
              pagination={false}
            />
          </TabPane>
        </Tabs>
      </Card>

      <Modal
        title="继续发布"
        open={continueModalVisible}
        onOk={handleContinue}
        onCancel={() => setContinueModalVisible(false)}
      >
        <div style={{ marginBottom: 16 }}>
          <label>灰度比例: </label>
          <InputNumber
            min={detail.grayPolicy.percentage}
            max={100}
            value={newPercentage}
            onChange={(value) => setNewPercentage(value || 0)}
            formatter={(value) => `${value}%`}
            style={{ width: 200 }}
          />
        </div>
        <p>当前灰度比例: {detail.grayPolicy.percentage}%</p>
      </Modal>
    </div>
  )
}
