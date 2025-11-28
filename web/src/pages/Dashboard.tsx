import React from 'react'
import { useQuery } from 'react-query'
import { 
  TrendingUp, 
  DollarSign, 
  Activity, 
  Clock,
  Zap,
  AlertCircle
} from 'lucide-react'

// Mock API calls - these would connect to the real API
const fetchBotStatus = async () => {
  // Simulate API call
  return {
    running: true,
    uptime: 3600,
    totalTrades: 42,
    successfulTrades: 38,
    failedTrades: 4,
    currentMode: 'paper'
  }
}

const fetchMetrics = async () => {
  return {
    systemMetrics: {
      uptime: 3600,
      memoryUsage: 45.2,
      cpuUsage: 12.5,
      activeConnections: 3
    },
    tradingMetrics: {
      totalTrades: 42,
      successRate: 90.5,
      averageExecutionTime: 1250,
      totalVolumeEth: 2.45,
      totalProfitEth: 0.08
    }
  }
}

export const Dashboard: React.FC = () => {
  const { data: botStatus, isLoading: statusLoading } = useQuery('botStatus', fetchBotStatus)
  const { data: metrics, isLoading: metricsLoading } = useQuery('metrics', fetchMetrics)

  const stats = [
    {
      name: 'Total Trades',
      value: botStatus?.totalTrades || 0,
      icon: TrendingUp,
      change: '+12%',
      changeType: 'positive'
    },
    {
      name: 'Success Rate',
      value: `${botStatus ? (botStatus.successfulTrades / botStatus.totalTrades * 100).toFixed(1) : 0}%`,
      icon: Activity,
      change: '+2.3%',
      changeType: 'positive'
    },
    {
      name: 'Total Volume',
      value: `${metrics?.tradingMetrics.totalVolumeEth || 0} ETH`,
      icon: DollarSign,
      change: '+18%',
      changeType: 'positive'
    },
    {
      name: 'Uptime',
      value: `${Math.floor((botStatus?.uptime || 0) / 3600)}h`,
      icon: Clock,
      change: 'Running',
      changeType: 'neutral'
    }
  ]

  return (
    <div>
      {/* Page Header */}
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-gray-900">Dashboard</h2>
        <p className="text-gray-600">Monitor your CryptoJackal bot performance and status</p>
      </div>

      {/* Status Alert */}
      {botStatus?.running && (
        <div className="mb-6 p-4 bg-success-50 border border-success-200 rounded-md flex items-center">
          <Zap className="w-5 h-5 text-success-600 mr-3" />
          <div>
            <h3 className="text-sm font-medium text-success-800">Bot is Active</h3>
            <p className="text-sm text-success-700">
              Running in {botStatus.currentMode === 'paper' ? 'Paper Trading' : 'Live'} mode
            </p>
          </div>
        </div>
      )}

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {stats.map((stat) => (
          <div key={stat.name} className="card">
            <div className="card-body">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">{stat.name}</p>
                  <p className="text-2xl font-bold text-gray-900">{stat.value}</p>
                </div>
                <div className={`p-3 rounded-full ${
                  stat.changeType === 'positive' ? 'bg-success-100' :
                  stat.changeType === 'negative' ? 'bg-danger-100' : 'bg-gray-100'
                }`}>
                  <stat.icon className={`w-6 h-6 ${
                    stat.changeType === 'positive' ? 'text-success-600' :
                    stat.changeType === 'negative' ? 'text-danger-600' : 'text-gray-600'
                  }`} />
                </div>
              </div>
              <div className={`mt-2 text-sm ${
                stat.changeType === 'positive' ? 'text-success-600' :
                stat.changeType === 'negative' ? 'text-danger-600' : 'text-gray-600'
              }`}>
                {stat.change} from last period
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* System Metrics */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-medium text-gray-900">System Performance</h3>
          </div>
          <div className="card-body space-y-4">
            <div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Memory Usage</span>
                <span className="font-medium">{metrics?.systemMetrics.memoryUsage}%</span>
              </div>
              <div className="mt-1 bg-gray-200 rounded-full h-2">
                <div 
                  className="bg-primary-600 h-2 rounded-full" 
                  style={{ width: `${metrics?.systemMetrics.memoryUsage}%` }}
                />
              </div>
            </div>
            <div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">CPU Usage</span>
                <span className="font-medium">{metrics?.systemMetrics.cpuUsage}%</span>
              </div>
              <div className="mt-1 bg-gray-200 rounded-full h-2">
                <div 
                  className="bg-success-600 h-2 rounded-full" 
                  style={{ width: `${metrics?.systemMetrics.cpuUsage}%` }}
                />
              </div>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-medium text-gray-900">Recent Activity</h3>
          </div>
          <div className="card-body">
            <div className="space-y-3">
              <div className="flex items-center text-sm">
                <div className="w-2 h-2 bg-success-500 rounded-full mr-3"></div>
                <span className="text-gray-600">Trade executed: 0.1 ETH → USDC</span>
                <span className="ml-auto text-gray-500">2 min ago</span>
              </div>
              <div className="flex items-center text-sm">
                <div className="w-2 h-2 bg-primary-500 rounded-full mr-3"></div>
                <span className="text-gray-600">New opportunity detected: DEMO token</span>
                <span className="ml-auto text-gray-500">5 min ago</span>
              </div>
              <div className="flex items-center text-sm">
                <div className="w-2 h-2 bg-warning-500 rounded-full mr-3"></div>
                <span className="text-gray-600">Gas price alert: High network congestion</span>
                <span className="ml-auto text-gray-500">12 min ago</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
