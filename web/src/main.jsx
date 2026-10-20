import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ConfigProvider, App as AntApp } from 'antd'
import './index.css'
import App from './App.jsx'

const theme = {
  token: {
    colorPrimary: '#0f9d76',
    borderRadius: 10,
    fontFamily: 'Inter, ui-sans-serif, system-ui, sans-serif',
  },
}

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <ConfigProvider theme={theme}>
      <AntApp>
        <App />
      </AntApp>
    </ConfigProvider>
  </StrictMode>,
)
