import { useState } from 'react'
import { App, Button, Segmented, Table, Tag } from 'antd'
import { api, formatDate } from '../api'

const kindLabels = {
  invalid_product: { color: 'red', label: 'Invalid product' },
  supplier_unavailable: { color: 'orange', label: 'Supplier unavailable' },
  update_interrupted: { color: 'gold', label: 'Update interrupted' },
}

// Problems that need attention. Most are resolved automatically
// (newer valid data, or the update completes); staff can also close them.
export default function ProblemsTable({ problems, onChanged }) {
  const { message } = App.useApp()
  const [filter, setFilter] = useState('open')

  async function resolve(problem) {
    try {
      await api.resolveProblem(problem.id)
      message.success('Problem marked as resolved')
      onChanged()
    } catch (err) {
      message.error(err.message)
    }
  }

  const shown = filter === 'all' ? problems : problems.filter((p) => p.status === filter)

  return (
    <>
      <Segmented
        className="mb-3"
        value={filter}
        onChange={setFilter}
        options={[
          { value: 'open', label: `Needs attention (${problems.filter((p) => p.status === 'open').length})` },
          { value: 'resolved', label: 'Resolved' },
          { value: 'all', label: 'All' },
        ]}
      />
      <Table
        rowKey="id"
        size="middle"
        dataSource={shown}
        pagination={{ pageSize: 8 }}
        columns={[
          {
            title: 'Kind',
            dataIndex: 'kind',
            width: 170,
            render: (k) => <Tag color={kindLabels[k]?.color}>{kindLabels[k]?.label || k}</Tag>,
          },
          { title: 'Update #', dataIndex: 'run_id', width: 90 },
          { title: 'Batch', dataIndex: 'batch', width: 70, render: (b) => b || '—' },
          { title: 'SKU', dataIndex: 'sku', width: 140, render: (s) => (s ? <code>{s}</code> : '—') },
          { title: 'Message', dataIndex: 'message' },
          {
            title: 'Status',
            width: 220,
            render: (_, p) =>
              p.status === 'open' ? (
                <Button size="small" onClick={() => resolve(p)}>
                  Mark resolved
                </Button>
              ) : (
                <span className="text-xs text-slate-500">
                  ✔ {p.resolution}
                  <br />
                  {formatDate(p.resolved_at)}
                </span>
              ),
          },
        ]}
      />
    </>
  )
}
