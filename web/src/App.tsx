import { Routes, Route, Navigate } from 'react-router-dom'
import AppLayout from './components/Layout'
import ReleaseList from './pages/Release/ReleaseList'
import ReleaseDetail from './pages/Release/ReleaseDetail'
import CreateRelease from './pages/Release/CreateRelease'
import AppManagement from './pages/App/AppManagement'

function App() {
  return (
    <Routes>
      <Route path="/" element={<AppLayout />}>
        <Route index element={<Navigate to="/releases" replace />} />
        <Route path="releases" element={<ReleaseList />} />
        <Route path="releases/create" element={<CreateRelease />} />
        <Route path="releases/:releaseId" element={<ReleaseDetail />} />
        <Route path="apps" element={<AppManagement />} />
      </Route>
    </Routes>
  )
}

export default App
