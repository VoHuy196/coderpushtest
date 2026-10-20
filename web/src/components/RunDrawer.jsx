import { useEffect, useState } from 'react'
import { Descriptions, Drawer, Segmented, Table, Typography } from 'antd'
import { api, formatDate } from '../api'
import { ActionTag, RunStatusTag } from './Tags'

// Side panel with the details and change log of one run.
export default function RunDrawer({ run, onClose }) {
  const [changes, setChanges] = useState([])
  const [filter, setFilter] = useState('all')

  // Reload the change log while the drawer is open (the run may still be running).
  useEffect(() => {
    if (!run) return
    api.runChanges(run.id).then(setChanges).catch(() => setChanges([]))
  }, [run])

  if (!run) return null

  const shown = filter === 'all' ? changes : changes.filter((c) => c.action === filter)
  const options = ['all', 'created', 'updated', 'withdrawn', 'skipped', 'invalid'].map((a) => ({
    value: a,
    label: a === 'all' ? `all (${changes.length})` : `${a} (${changes.filter((c) => c.action === a).length})`,
  }))

  return (
    <Drawer open width={860} onClose={onClose} title={`Update #${run.id} — ${run.supplier_update_id}`}>
      <Descriptions size="small" column={2} bordered>
        <Descriptions.Item label="Status">
          <RunStatusTag status={run.status} />
        </Descriptions.Item>
        <Descriptions.Item label="Batches saved">
          {run.next_batch - 1} / {run.total_batches || '?'}
        </Descriptions.Item>
        <Descriptions.Item label="Started">{formatDate(run.started_at)}</Descriptions.Item>
        <Descriptions.Item label="Finished">{formatDate(run.finished_at)}</Descriptions.Item>
        <Descriptions.Item label="Open problems">{run.open_problems}</Descriptions.Item>
        <Descriptions.Item label="Last error">{run.last_error || '—'}</Descriptions.Item>
      </Descriptions>

      <Typography.Title level={5} className="!mt-6">
        What changed
      </Typography.Title>
      <Segmented options={options} value={filter} onChange={setFilter} className="mb-3" />
      <Table
        rowKey="id"
        size="small"
        dataSource={shown}
        pagination={{ pageSize: 12 }}
        columns={[
          { title: 'Batch', dataIndex: 'batch', width: 70 },
          { title: 'SKU', dataIndex: 'sku', width: 150, render: (v) => <code>{v || '(none)'}</code> },
          { title: 'Action', dataIndex: 'action', width: 100, render: (a) => <ActionTag action={a} /> },
          { title: 'Detail', dataIndex: 'detail' },
        ]}
      />
    </Drawer>
  )
}
