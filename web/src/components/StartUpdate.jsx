import { useEffect, useState } from 'react'
import { App, Button, Card, Select, Space, Tooltip, Typography } from 'antd'
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
    <Card
      title="Start an update"
      className="h-full border-slate-200 shadow-sm rounded-xl"
      extra={
        <Tooltip title="Refresh supplier updates list">
          <Button
            id="reload-updates"
            type="text"
            size="small"
            icon={<ReloadOutlined className="text-slate-500 hover:text-emerald-600" />}
            onClick={loadUpdates}
          >
            Refresh
          </Button>
        </Tooltip>
      }
    >
      <div className="flex flex-col gap-4">
        <div>
          <Typography.Text type="secondary" className="text-xs font-medium block mb-2">
            Select a published update from supplier:
          </Typography.Text>
          <Select
            id="update-select"
            className="w-full"
            placeholder="Choose an update to import"
            value={selected}
            onChange={setSelected}
            popupMatchSelectWidth={false}
            options={updates.map((u) => ({
              value: u.id,
              label: `${u.id} — ${u.description} (${u.total_batches} batches)`,
            }))}
          />
        </div>

        {error && (
          <Typography.Text type="danger" className="text-xs">
            {error}
          </Typography.Text>
        )}

        <Button
          id="start-update"
          type="primary"
          icon={<CloudDownloadOutlined />}
          size="middle"
          block
          disabled={!selected || disabled}
          loading={loading}
          onClick={start}
          className="mt-1 shadow-sm font-medium"
        >
          Start update
        </Button>

        {disabled && (
          <div className="bg-amber-50 border border-amber-200 rounded-lg p-2.5">
            <Typography.Text type="warning" className="text-xs block leading-relaxed">
              ⚠️ An update is currently in progress or interrupted. Please finish, resume, or abandon it before starting a new one.
            </Typography.Text>
          </div>
        )}
      </div>
    </Card>
  )
}
