import { useEffect, useState } from 'react'
import { App, Button, Card, Select, Space, Typography } from 'antd'
import { CloudDownloadOutlined, ReloadOutlined } from '@ant-design/icons'
import { api } from '../api'

// Lets staff pick a published supplier update and start importing it.
export default function StartUpdate({ disabled, onStarted }) {
  const { message } = App.useApp()
  const [updates, setUpdates] = useState([])
  const [selected, setSelected] = useState()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function loadUpdates() {
    try {
      const list = await api.supplierUpdates()
      setUpdates(list)
      setError('')
    } catch (err) {
      setError(err.message)
    }
  }

  useEffect(() => {
    loadUpdates()
  }, [])

  async function start() {
    setLoading(true)
    try {
      const run = await api.startRun(selected)
      message.success(`Update #${run.id} started`)
      onStarted()
    } catch (err) {
      message.error(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card title="Start an update" className="h-full">
      <Space direction="vertical" className="w-full">
        <Typography.Text type="secondary">Updates published by the supplier:</Typography.Text>
        <Space.Compact className="w-full">
          <Select
            id="update-select"
            className="w-full"
            placeholder="Choose an update"
            value={selected}
            onChange={setSelected}
            options={updates.map((u) => ({
              value: u.id,
              label: `${u.id} — ${u.description} (${u.total_batches} batches)`,
            }))}
          />
          <Button id="reload-updates" icon={<ReloadOutlined />} onClick={loadUpdates} />
        </Space.Compact>
        {error && <Typography.Text type="danger">{error}</Typography.Text>}
        <Button
          id="start-update"
          type="primary"
          icon={<CloudDownloadOutlined />}
          block
          disabled={!selected || disabled}
          loading={loading}
          onClick={start}
        >
          Start update
        </Button>
        {disabled && (
          <Typography.Text type="secondary" className="text-xs">
            Finish, resume or abandon the current update before starting a new one.
          </Typography.Text>
        )}
      </Space>
    </Card>
  )
}
