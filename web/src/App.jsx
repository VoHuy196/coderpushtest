import { useEffect, useState } from 'react'
import { Alert, Badge, Card, Layout, Row, Col, Space, Switch, Tabs, Typography, Button } from 'antd'
import {
  AppstoreOutlined,
  CloudDownloadOutlined,
  ExclamationCircleOutlined,
  HistoryOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
import { api } from './api'
import StartUpdate from './components/StartUpdate'
import RunsTable from './components/RunsTable'
import RunDrawer from './components/RunDrawer'
import ProblemsTable from './components/ProblemsTable'
import ProductsTable from './components/ProductsTable'

export default function App() {
  const [summary, setSummary] = useState({ active_products: 0, withdrawn_products: 0, open_problems: 0 })
  const [runs, setRuns] = useState([])
  const [products, setProducts] = useState([])
  const [problems, setProblems] = useState([])
  const [outage, setOutage] = useState(false)
  const [activeRun, setActiveRun] = useState(null)
  const [activeTab, setActiveTab] = useState('runs')

  async function loadData() {
    try {
      const [sumData, runsData, prodData, probData, outData] = await Promise.all([
        api.summary(),
        api.runs(),
        api.products(),
        api.problems(),
        api.getOutage().catch(() => ({ enabled: false })),
      ])
      setSummary(sumData)
      setRuns(runsData || [])
      setProducts(prodData || [])
      setProblems(probData || [])
      setOutage(!!outData.enabled)

      // keep drawer in sync if open
      if (activeRun) {
        const found = (runsData || []).find((r) => r.id === activeRun.id)
        if (found) setActiveRun(found)
      }
    } catch (err) {
      console.error('Failed to load data:', err)
    }
  }

  useEffect(() => {
    loadData()
    const interval = setInterval(loadData, 2500)
    return () => clearInterval(interval)
  }, [activeRun?.id])

  async function toggleOutage(checked) {
    try {
      await api.setOutage(checked)
      setOutage(checked)
    } catch (err) {
      console.error('Failed to toggle outage:', err)
    }
  }

  const hasRunning = runs.some((r) => r.status === 'running' || r.status === 'interrupted')

  return (
    <Layout className="min-h-screen bg-slate-50">
      {/* Top Header */}
      <header className="hero text-white px-6 py-6 shadow-md">
        <div className="max-w-7xl mx-auto flex flex-wrap items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <span className="pulse-dot"></span>
              <Typography.Title level={3} className="!text-white !mb-0 tracking-tight">
                Supplier Catalog Hub
              </Typography.Title>
            </div>
            <Typography.Paragraph className="!text-slate-300 !mb-0 text-sm mt-1">
              Automated delta imports with checkpointing, versioning & problem triage.
            </Typography.Paragraph>
          </div>

          {/* Quick Demo Outage Toggle */}
          <div className="bg-slate-900/60 backdrop-blur-md rounded-xl p-3 border border-slate-700/60 flex items-center gap-3">
            <ThunderboltOutlined className={outage ? 'text-amber-400 text-lg' : 'text-slate-400 text-lg'} />
            <div className="text-xs">
              <div className="font-semibold text-slate-200">Simulate Supplier Outage (503)</div>
              <div className="text-slate-400">Used for testing interruption & recovery</div>
            </div>
            <Switch
              checked={outage}
              onChange={toggleOutage}
              className={outage ? 'bg-amber-500' : ''}
              checkedChildren="OFFLINE"
              unCheckedChildren="ONLINE"
            />
          </div>
        </div>
      </header>

      {/* Main Container */}
      <main className="max-w-7xl mx-auto p-6 w-full space-y-6">
        {/* Outage Warning Alert */}
        {outage && (
          <Alert
            message="Simulated Supplier Outage is Active"
            description="All batch fetch requests will return HTTP 503. Ongoing updates will be safely interrupted without losing committed data."
            type="warning"
            showIcon
            className="border-amber-400 bg-amber-50"
          />
        )}

        {/* Dashboard Stat Cards */}
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={8}>
            <Card className="stat-card border-slate-200 shadow-sm rounded-xl">
              <div className="text-xs font-semibold uppercase tracking-wider text-slate-400">Active Products</div>
              <div className="text-3xl font-bold text-emerald-600 mt-1">{summary.active_products}</div>
              <div className="text-xs text-slate-500 mt-1">Currently listed in local catalog</div>
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card className="stat-card border-slate-200 shadow-sm rounded-xl">
              <div className="text-xs font-semibold uppercase tracking-wider text-slate-400">Withdrawn (Tombstones)</div>
              <div className="text-3xl font-bold text-slate-600 mt-1">{summary.withdrawn_products}</div>
              <div className="text-xs text-slate-500 mt-1">Preserved to block out-of-order restoration</div>
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card className="stat-card border-slate-200 shadow-sm rounded-xl">
              <div className="text-xs font-semibold uppercase tracking-wider text-slate-400">Problems Needing Attention</div>
              <div className="text-3xl font-bold text-red-500 mt-1">
                {summary.open_problems}
              </div>
              <div className="text-xs text-slate-500 mt-1">Validation errors or network interruptions</div>
            </Card>
          </Col>
        </Row>

        {/* Control & Run Area */}
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={8}>
            <StartUpdate disabled={hasRunning} onStarted={loadData} />
          </Col>

          <Col xs={24} lg={16}>
            <Card
              className="h-full border-slate-200 shadow-sm rounded-xl"
              title="Catalog Updates & Triage"
              extra={
                <Button icon={<ReloadOutlined />} onClick={loadData} size="small">
                  Refresh
                </Button>
              }
            >
              <Tabs
                activeKey={activeTab}
                onChange={setActiveTab}
                items={[
                  {
                    key: 'runs',
                    label: (
                      <span>
                        <HistoryOutlined />
                        Update Runs ({runs.length})
                      </span>
                    ),
                    children: <RunsTable runs={runs} onOpen={(run) => setActiveRun(run)} onChanged={loadData} />,
                  },
                  {
                    key: 'problems',
                    label: (
                      <span>
                        <Badge count={summary.open_problems} size="small" offset={[8, 0]}>
                          <ExclamationCircleOutlined />
                          Problems
                        </Badge>
                      </span>
                    ),
                    children: <ProblemsTable problems={problems} onChanged={loadData} />,
                  },
                  {
                    key: 'products',
                    label: (
                      <span>
                        <AppstoreOutlined />
                        Local Catalog ({products.length})
                      </span>
                    ),
                    children: <ProductsTable products={products} />,
                  },
                ]}
              />
            </Card>
          </Col>
        </Row>
      </main>

      {/* Details Drawer */}
      <RunDrawer run={activeRun} onClose={() => setActiveRun(null)} />
    </Layout>
  )
}
