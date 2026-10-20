import { Tag } from 'antd'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  PauseCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons'

const runStatus = {
  running: { color: 'processing', icon: <SyncOutlined spin />, label: 'Running' },
  interrupted: { color: 'warning', icon: <PauseCircleOutlined />, label: 'Interrupted' },
  completed: { color: 'success', icon: <CheckCircleOutlined />, label: 'Completed' },
  abandoned: { color: 'default', icon: <CloseCircleOutlined />, label: 'Abandoned' },
}

export function RunStatusTag({ status }) {
  const s = runStatus[status] || { color: 'default', label: status }
  return (
    <Tag color={s.color} icon={s.icon}>
      {s.label}
    </Tag>
  )
}

const actionColors = {
  created: 'green',
  updated: 'blue',
  withdrawn: 'volcano',
  skipped: 'default',
  invalid: 'red',
}

export function ActionTag({ action }) {
  return <Tag color={actionColors[action]}>{action}</Tag>
}

export function ProductStatusTag({ status }) {
  return status === 'active' ? <Tag color="green">active</Tag> : <Tag color="volcano">withdrawn</Tag>
}
