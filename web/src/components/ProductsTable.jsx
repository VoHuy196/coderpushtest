import { useState } from 'react'
import { Input, Segmented, Table } from 'antd'
import { SearchOutlined } from '@ant-design/icons'
import { formatDate, formatPrice } from '../api'
import { ProductStatusTag } from './Tags'

// The local copy of the supplier catalog.
export default function ProductsTable({ products }) {
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState('all')

  const text = search.toLowerCase()
  const shown = products.filter(
    (p) =>
      (status === 'all' || p.status === status) &&
      (p.sku.toLowerCase().includes(text) || p.name.toLowerCase().includes(text)),
  )

  return (
    <>
      <div className="mb-3 flex flex-wrap gap-3">
        <Input
          id="product-search"
          allowClear
          prefix={<SearchOutlined />}
          placeholder="Search SKU or name"
          className="!w-64"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Segmented
          value={status}
          onChange={setStatus}
          options={[
            { value: 'all', label: 'All' },
            { value: 'active', label: 'Active' },
            { value: 'withdrawn', label: 'Withdrawn' },
          ]}
        />
      </div>
      <Table
        rowKey="sku"
        size="middle"
        dataSource={shown}
        pagination={{ pageSize: 10 }}
        rowClassName={(p) => (p.status === 'withdrawn' ? 'opacity-60' : '')}
        columns={[
          { title: 'SKU', dataIndex: 'sku', width: 160, render: (v) => <code>{v}</code> },
          { title: 'Name', dataIndex: 'name', render: (v) => v || <i className="text-slate-400">unknown</i> },
          { title: 'Price', width: 130, render: (_, p) => formatPrice(p.price_cents, p.currency) },
          { title: 'Status', dataIndex: 'status', width: 110, render: (s) => <ProductStatusTag status={s} /> },
          { title: 'Supplier updated', dataIndex: 'supplier_updated_at', width: 190, render: formatDate },
          { title: 'Last update #', dataIndex: 'last_run_id', width: 110 },
        ]}
      />
    </>
  )
}
