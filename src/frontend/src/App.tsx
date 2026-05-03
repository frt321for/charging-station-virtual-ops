import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { DevHealthPage } from './pages/dev-health/DevHealthPage'
import { OperationsPage } from './pages/operations/OperationsPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<OperationsPage />} />
        <Route path="/operations" element={<OperationsPage />} />
        <Route path="/dev/health" element={<DevHealthPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
