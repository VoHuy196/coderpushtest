import { App, Button, Popconfirm, Progress, Space, Table, Tooltip } from 'antd'
import { api, formatDate } from '../api'
import { RunStatusTag } from './Tags'

// Shows all update runs with progress and Resume / Abandon actions.
export default function RunsTable({ runs, onOpen, onChanged }) {
  const { message } = App.useApp()

  async function resume(run) {
    try {
      await api.resumeRun(run.id)
      message.success(`Resuming update #${run.id} from batch ${run.next_batch}`)
      onChanged()
    } catch (err) {
      message.error(err.message)
    }
  }

  async function abandon(run) {
    try {
      await api.abandonRun(run.id)
      message.info(`Update #${run.id} abandoned`)
      onChanged()
    } catch (err) {
      message.error(err.message)
    }
  }

  const columns = [
    { title: '#', dataIndex: 'id', width: 60 },
    { title: 'Supplier update', dataIndex: 'supplier_update_id' },
    {
      title: 'Status',
      dataIndex: 'status',
      render: (status, run) => (
        <Tooltip title={run.last_error}>
          <span>
            <RunStatusTag status={status} />
          </span>
        </Tooltip>
      ),
    },
    {
      title: 'Batches',
      width: 180,
      render: (_, run) => {
        const done = run.next_batch - 1
        const total = run.total_batches
        const percent = total ? Math.round((done / total) * 100) : 0
        return (
          <Space>
            <Progress
              percent={run.status === 'completed' ? 100 : percent}
              size="small"
              status={run.status === 'interrupted' ? 'exception' : run.status === 'running' ? 'active' : undefined}
              showInfo={false}
              style={{ width: 80 }}
            />
            <span className="text-xs text-slate-500">
              {done}/{total || '?'}
            </span>
          </Space>
        )
      },
    },
    {
      title: 'Changes',
      render: (_, run) => (
        <span className="text-xs">
          <b className="text-emerald-600">+{run.counts.created}</b>{' '}
          <b className="text-blue-600">~{run.counts.updated}</b>{' '}
          <b className="text-orange-600">−{run.counts.withdrawn}</b>{' '}
          <span className="text-slate-400">skip {run.counts.skipped}</span>{' '}
          {run.counts.invalid > 0 && <span className="text-red-500">invalid {run.counts.invalid}</span>}
        </span>
      ),
    },
    { title: 'Started', dataIndex: 'started_at', render: formatDate },
    { title: 'Finished', dataIndex: 'finished_at', render: formatDate },
    {
      title: '',
      render: (_, run) => (
        <Space size="small" wrap onClick={(e) => e.stopPropagation()}>
          <Button size="small" onClick={() => onOpen(run)}>
            Details
          </Button>
          {run.status === 'interrupted' && (
            <>
              <Button id={`resume-${run.id}`} size="small" type="primary" onClick={() => resume(run)}>
                Resume
              </Button>
              <Popconfirm
                title="Abandon this update?"
                description="Saved batches stay. The update will not be marked complete."
                onConfirm={() => abandon(run)}
              >
                <Button id={`abandon-${run.id}`} size="small" danger>
                  Abandon
                </Button>
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ]

  return (
    <Table
      rowKey="id"
      size="middle"
      columns={columns}
      dataSource={runs}
      scroll={{ x: 750 }}
      pagination={{ pageSize: 8 }}
      onRow={(run) => ({ onClick: () => onOpen(run), className: 'cursor-pointer' })}
    />
  )
}
