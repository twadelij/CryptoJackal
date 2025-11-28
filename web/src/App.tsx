import React from 'react'
import { Routes, Route } from 'react-router-dom'
import { Layout } from './components/Layout'
import { Dashboard } from './pages/Dashboard'
import { Trading } from './pages/Trading'
import { Discovery } from './pages/Discovery'
import { PaperTrading } from './pages/PaperTrading'
import { Settings } from './pages/Settings'

function App() {
  return (
    <div className="min-h-screen bg-gray-50">
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/trading" element={<Trading />} />
          <Route path="/discovery" element={<Discovery />} />
          <Route path="/paper-trading" element={<PaperTrading />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </Layout>
    </div>
  )
}

export default App
