// home/web/src/App.jsx
import React, { useState, useEffect, useCallback } from 'react';

const StatCard = ({ title, value, children, status = 'normal', size = 'normal' }) => {
  const statusColors = {
    normal: 'bg-gray-800',
    warning: 'bg-yellow-900',
    critical: 'bg-red-900',
    offline: 'bg-gray-700'
  };

  const sizeClasses = {
    small: 'p-4',
    normal: 'p-6'
  };

  return (
    <div className={`${statusColors[status]} ${sizeClasses[size]} rounded-lg shadow-lg transition-colors duration-300`}>
      <h3 className="text-lg font-semibold text-gray-400">{title}</h3>
      <p className={`${size === 'small' ? 'text-xl' : 'text-3xl'} font-bold text-white mt-2`}>{value}</p>
      {children}
    </div>
  );
};

const ProgressBar = ({ value, max, showPercentage = false, size = 'normal' }) => {
  const percentage = Math.min((value / max) * 100, 100);
  
  let colorClass = 'bg-green-500';
  if (percentage > 80) colorClass = 'bg-red-500';
  else if (percentage > 60) colorClass = 'bg-yellow-500';
  else if (percentage > 40) colorClass = 'bg-blue-500';

  const heightClass = size === 'small' ? 'h-2' : 'h-4';

  return (
    <div className={`w-full bg-gray-700 rounded-full ${heightClass} mt-2`}>
      <div
        className={`${colorClass} ${heightClass} rounded-full transition-all duration-500 ease-out`}
        style={{ width: `${percentage}%` }}
      ></div>
      {showPercentage && (
        <div className="text-sm text-gray-400 mt-1">
          {percentage.toFixed(1)}%
        </div>
      )}
    </div>
  );
};

const ConnectionStatus = ({ isConnected, lastUpdate, agentCount, onlineCount }) => (
  <div className="flex items-center space-x-4 mb-4">
    <div className="flex items-center space-x-2">
      <div className={`w-3 h-3 rounded-full ${isConnected ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`}></div>
      <span className={`text-sm ${isConnected ? 'text-green-400' : 'text-red-400'}`}>
        {isConnected ? 'Connected' : 'Disconnected'}
      </span>
    </div>
    <div className="text-sm text-gray-400">
      Agents: <span className="text-green-400">{onlineCount}</span>/<span className="text-blue-400">{agentCount}</span> online
    </div>
    {lastUpdate && (
      <span className="text-xs text-gray-500">
        Last update: {new Date(lastUpdate).toLocaleTimeString()}
      </span>
    )}
  </div>
);

const DockerStatsTable = ({ dockerStats }) => {
  if (!dockerStats || dockerStats.length === 0) return null;

  return (
    <div className="bg-gray-800 p-6 rounded-lg shadow-lg">
      <h3 className="text-lg font-semibold text-gray-400 mb-4">Docker Containers ({dockerStats.length})</h3>
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-gray-700">
              <th className="p-3 text-gray-400">Container</th>
              <th className="p-3 text-gray-400">CPU %</th>
              <th className="p-3 text-gray-400">Memory</th>
              <th className="p-3 text-gray-400">Mem %</th>
              <th className="p-3 text-gray-400">Net I/O</th>
              <th className="p-3 text-gray-400">PIDs</th>
            </tr>
          </thead>
          <tbody>
            {dockerStats.map((container, index) => (
              <tr key={`${container.container_id}-${index}`} className="border-b border-gray-700 hover:bg-gray-700 transition-colors">
                <td className="p-3">
                  <div>
                    <div className="text-blue-400 font-mono text-xs">{container.name}</div>
                    <div className="text-gray-500 text-xs">{container.container_id.substring(0, 12)}</div>
                  </div>
                </td>
                <td className="p-3">
                  <span className={`${parseFloat(container.cpu_percent) > 50 ? 'text-red-400' : parseFloat(container.cpu_percent) > 25 ? 'text-yellow-400' : 'text-green-400'}`}>
                    {container.cpu_percent}
                  </span>
                </td>
                <td className="p-3 text-gray-300">
                  <div>{container.memory_usage}</div>
                  {container.memory_limit && <div className="text-xs text-gray-500">/ {container.memory_limit}</div>}
                </td>
                <td className="p-3">
                  <span className={`${parseFloat(container.memory_percent) > 80 ? 'text-red-400' : parseFloat(container.memory_percent) > 60 ? 'text-yellow-400' : 'text-green-400'}`}>
                    {container.memory_percent}
                  </span>
                </td>
                <td className="p-3 text-gray-300 text-xs">{container.network_io}</td>
                <td className="p-3 text-gray-300">{container.pids}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

const AgentSummaryCard = ({ agent, onClick, isSelected }) => {
  const getStatusColor = (isOnline) => isOnline ? 'border-green-500' : 'border-red-500';
  
  return (
    <div 
      className={`bg-gray-800 p-4 rounded-lg cursor-pointer transition-all duration-200 border-2 ${
        isSelected ? 'border-blue-500 bg-gray-700' : getStatusColor(agent.is_online)
      } hover:bg-gray-700`}
      onClick={() => onClick(agent.agent_id)}
    >
      <div className="flex justify-between items-start mb-2">
        <div>
          <h4 className="text-lg font-semibold text-white">{agent.agent_id}</h4>
          <p className="text-sm text-gray-400">{agent.hostname}</p>
        </div>
        <span className={`text-xs px-2 py-1 rounded ${
          agent.is_online ? 'bg-green-700 text-green-200' : 'bg-red-700 text-red-200'
        }`}>
          {agent.is_online ? 'Online' : 'Offline'}
        </span>
      </div>
      
      {agent.is_online && (
        <div className="grid grid-cols-2 gap-2 text-xs">
          <div>
            <span className="text-gray-400">CPU:</span>
            <span className={`ml-1 ${agent.cpu_usage > 80 ? 'text-red-400' : 'text-green-400'}`}>
              {agent.cpu_usage.toFixed(1)}%
            </span>
          </div>
          <div>
            <span className="text-gray-400">RAM:</span>
            <span className={`ml-1 ${agent.memory_usage > 80 ? 'text-red-400' : 'text-green-400'}`}>
              {agent.memory_usage.toFixed(1)}%
            </span>
          </div>
          <div>
            <span className="text-gray-400">Proc:</span>
            <span className="ml-1 text-blue-400">{agent.process_count}</span>
          </div>
          <div>
            <span className="text-gray-400">Docker:</span>
            <span className="ml-1 text-purple-400">{agent.docker_count}</span>
          </div>
        </div>
      )}
    </div>
  );
};

const AgentView = ({ agent, formatBytes, formatUptime }) => {
  if (!agent) {
    return (
      <div className="text-center text-gray-500 py-8">
        <h2 className="text-2xl font-bold mb-2">Agent Not Found</h2>
        <p>The selected agent is not available or has been disconnected.</p>
      </div>
    );
  }

  const isOnline = new Date() - new Date(agent.last_seen) < 120000;
  const systemStatus = getSystemStatus(agent);

  return (
    <div className="space-y-6">
      {/* Agent Header */}
      <div className="bg-gray-800 p-6 rounded-lg">
        <div className="flex justify-between items-start">
          <div>
            <h2 className="text-2xl font-bold text-white">{agent.agent_id}</h2>
            <p className="text-gray-400">{agent.hostname}</p>
            <p className="text-sm text-gray-500 mt-1">
              Uptime: {formatUptime(agent.uptime)}
            </p>
            {agent.system_info && (
              <div className="mt-2 text-sm text-gray-400">
                <p>Total Processes: <span className="text-blue-400">{agent.system_info.total_processes}</span></p>
                {agent.system_info.kernel_version && (
                  <p>Kernel: <span className="text-green-400">{agent.system_info.kernel_version}</span></p>
                )}
              </div>
            )}
          </div>
          <div className="text-right">
            <span className={`px-3 py-1 rounded text-sm ${
              isOnline ? 'bg-green-700 text-green-200' : 'bg-red-700 text-red-200'
            }`}>
              {isOnline ? 'Online' : 'Offline'}
            </span>
            <p className="text-xs text-gray-500 mt-1">
              Last seen: {new Date(agent.last_seen).toLocaleTimeString()}
            </p>
          </div>
        </div>
      </div>

      {!isOnline && (
        <div className="bg-red-900 border border-red-700 p-4 rounded-lg">
          <h3 className="text-red-200 font-semibold">Agent Offline</h3>
          <p className="text-red-300 text-sm">This agent hasn't reported metrics recently. Data shown is from the last known state.</p>
        </div>
      )}

      {/* System Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard 
          title="CPU Usage" 
          value={`${agent.cpu_usage.toFixed(1)}%`}
          status={!isOnline ? 'offline' : agent.cpu_usage > 80 ? 'critical' : agent.cpu_usage > 60 ? 'warning' : 'normal'}
        >
          <ProgressBar value={agent.cpu_usage} max={100} />
        </StatCard>
        
        <StatCard 
          title="Memory" 
          value={`${agent.memory?.usedPercent?.toFixed(1) || 0}%`}
          status={!isOnline ? 'offline' : (agent.memory?.usedPercent || 0) > 90 ? 'critical' : (agent.memory?.usedPercent || 0) > 75 ? 'warning' : 'normal'}
        >
          <ProgressBar value={agent.memory?.used || 0} max={agent.memory?.total || 1} />
          <div className="text-sm text-gray-400 mt-2">
            {formatBytes(agent.memory?.used || 0)} / {formatBytes(agent.memory?.total || 0)}
          </div>
        </StatCard>

        <StatCard 
          title="Disk Space" 
          value={`${agent.disk?.usedPercent?.toFixed(1) || 0}%`}
          status={!isOnline ? 'offline' : (agent.disk?.usedPercent || 0) > 90 ? 'critical' : (agent.disk?.usedPercent || 0) > 80 ? 'warning' : 'normal'}
        >
          <ProgressBar value={agent.disk?.used || 0} max={agent.disk?.total || 1} />
          <div className="text-sm text-gray-400 mt-2">
            {formatBytes(agent.disk?.used || 0)} / {formatBytes(agent.disk?.total || 0)}
          </div>
        </StatCard>

        <StatCard 
          title="System Status" 
          value={systemStatus.charAt(0).toUpperCase() + systemStatus.slice(1)}
          status={!isOnline ? 'offline' : systemStatus}
        >
          <div className="text-sm text-gray-400 mt-2">
            {agent.processes?.length || 0} top processes
          </div>
        </StatCard>
      </div>

      {/* Docker Stats */}
      {agent.docker_stats && agent.docker_stats.length > 0 && (
        <DockerStatsTable dockerStats={agent.docker_stats} />
      )}

      {/* Network Stats */}
      {agent.network && agent.network.length > 0 && (
        <div className="bg-gray-800 p-6 rounded-lg">
          <h3 className="text-lg font-semibold text-gray-400 mb-4">Network Usage</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-gray-400">Total Sent</p>
              <p className="text-2xl font-bold text-blue-400">
                {formatBytes(agent.network.reduce((sum, iface) => sum + (iface.bytesSent || 0), 0))}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-400">Total Received</p>
              <p className="text-2xl font-bold text-green-400">
                {formatBytes(agent.network.reduce((sum, iface) => sum + (iface.bytesRecv || 0), 0))}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Process List */}
      <div className="bg-gray-800 p-6 rounded-lg">
        <h3 className="text-lg font-semibold text-gray-400 mb-4">
          Top Processes ({agent.processes?.length || 0})
        </h3>
        {agent.processes && agent.processes.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="border-b border-gray-700">
                  <th className="p-3 text-gray-400">PID</th>
                  <th className="p-3 text-gray-400">Process Name</th>
                  <th className="p-3 text-gray-400">CPU %</th>
                  <th className="p-3 text-gray-400">Memory %</th>
                  <th className="p-3 text-gray-400">Command</th>
                </tr>
              </thead>
              <tbody>
                {agent.processes.slice(0, 20).map((process, index) => (
                  <tr key={`${process.pid}-${index}`} className="border-b border-gray-700 hover:bg-gray-700 transition-colors">
                    <td className="p-3 text-blue-400">{process.pid}</td>
                    <td className="p-3 font-mono text-sm">{process.name}</td>
                    <td className="p-3">
                      <span className={`${process.cpu_percent > 50 ? 'text-red-400' : process.cpu_percent > 25 ? 'text-yellow-400' : 'text-green-400'}`}>
                        {(process.cpu_percent || 0).toFixed(1)}%
                      </span>
                    </td>
                    <td className="p-3">
                      <span className={`${process.memory_percent > 10 ? 'text-red-400' : process.memory_percent > 5 ? 'text-yellow-400' : 'text-green-400'}`}>
                        {(process.memory_percent || 0).toFixed(1)}%
                      </span>
                    </td>
                    <td className="p-3 text-gray-400 text-sm max-w-xs truncate" title={process.command}>
                      {process.command}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center text-gray-500 py-8">
            No process data available
          </div>
        )}
      </div>

      {/* Agent Info */}
      {agent.agent_info && (
        <div className="bg-gray-800 p-6 rounded-lg">
          <h3 className="text-lg font-semibold text-gray-400 mb-4">Agent Information</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <span className="text-gray-400">Version:</span>
              <span className="ml-2 text-blue-400">{agent.agent_info.version}</span>
            </div>
            <div>
              <span className="text-gray-400">Go Version:</span>
              <span className="ml-2 text-green-400">{agent.agent_info.go_version}</span>
            </div>
            <div>
              <span className="text-gray-400">Goroutines:</span>
              <span className="ml-2 text-yellow-400">{agent.agent_info.num_goroutines}</span>
            </div>
            <div>
              <span className="text-gray-400">Memory Alloc:</span>
              <span className="ml-2 text-purple-400">{formatBytes(agent.agent_info.mem_stats?.alloc || 0)}</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const getSystemStatus = (agent) => {
  if (!agent) return 'normal';
  
  const isOnline = new Date() - new Date(agent.last_seen) < 120000;
  if (!isOnline) return 'offline';
  
  if (agent.cpu_usage > 90 || 
      (agent.memory?.usedPercent || 0) > 95 || 
      (agent.disk?.usedPercent || 0) > 95) {
    return 'critical';
  }
  
  if (agent.cpu_usage > 70 || 
      (agent.memory?.usedPercent || 0) > 80 || 
      (agent.disk?.usedPercent || 0) > 80) {
    return 'warning';
  }
  
  return 'normal';
};

function App() {
  const [multiAgentData, setMultiAgentData] = useState(null);
  const [socket, setSocket] = useState(null);
  const [isConnected, setIsConnected] = useState(false);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);
  const [lastUpdate, setLastUpdate] = useState(null);
  const [selectedAgent, setSelectedAgent] = useState('overview');

  const formatBytes = useCallback((bytes) => {
    if (bytes === 0) return '0 B';
    
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return parseFloat((bytes / Math.pow(1024, i)).toFixed(2)) + ' ' + sizes[i];
  }, []);

  const formatUptime = useCallback((seconds) => {
    const days = Math.floor(seconds / (24 * 3600));
    const hours = Math.floor((seconds % (24 * 3600)) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) return `${days}d ${hours}h ${minutes}m`;
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
  }, []);

  const connectWebSocket = useCallback(() => {
    if (socket) {
      socket.close();
    }

    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/api/ws`;

    console.log('Connecting to WebSocket:', wsUrl);
    const newSocket = new WebSocket(wsUrl);
    setSocket(newSocket);

    newSocket.onopen = () => {
      console.log('WebSocket connected');
      setIsConnected(true);
      setReconnectAttempts(0);
    };

    newSocket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setMultiAgentData(data);
        setLastUpdate(data.timestamp || new Date().toISOString());
      } catch (error) {
        console.error('Error parsing WebSocket message:', error);
      }
    };

    newSocket.onclose = (event) => {
      console.log('WebSocket disconnected', event.code, event.reason);
      setIsConnected(false);
      
      if (reconnectAttempts < 10) {
        const timeout = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
        console.log(`Reconnecting in ${timeout}ms...`);
        setTimeout(() => {
          setReconnectAttempts(prev => prev + 1);
          connectWebSocket();
        }, timeout);
      }
    };

    newSocket.onerror = (error) => {
      console.error('WebSocket error:', error);
      setIsConnected(false);
    };
  }, [socket, reconnectAttempts]);

  useEffect(() => {
    connectWebSocket();

    return () => {
      if (socket) {
        socket.close();
      }
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (multiAgentData && selectedAgent === 'overview' && multiAgentData.summary && multiAgentData.summary.length === 1) {
      setSelectedAgent(multiAgentData.summary[0].agent_id);
    }
  }, [multiAgentData, selectedAgent]);

  if (!multiAgentData) {
    return (
      <div className="min-h-screen bg-gray-900 text-white flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-blue-500 mx-auto mb-4"></div>
          <h1 className="text-4xl font-bold mb-2">Connecting to VPS Monitor...</h1>
          <p className="text-gray-400">
            {isConnected ? 
              'Waiting for agent data...' : 
              `Connection attempts: ${reconnectAttempts + 1}`
            }
          </p>
          {reconnectAttempts > 3 && (
            <button 
              onClick={connectWebSocket}
              className="mt-4 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
            >
              Retry Connection
            </button>
          )}
        </div>
      </div>
    );
  }

  const agentCount = multiAgentData.summary ? multiAgentData.summary.length : 0;
  const onlineCount = multiAgentData.summary ? multiAgentData.summary.filter(agent => agent.is_online).length : 0;

  return (
    <div className="min-h-screen bg-gray-900 text-white p-8">
      <div className="max-w-7xl mx-auto">
        <header className="mb-8">
          <div className="flex justify-between items-start">
            <div>
              <h1 className="text-4xl font-bold">VPS Monitor</h1>
              <p className="text-gray-400">
                {agentCount === 0 ? 'No agents connected' : 
                 agentCount === 1 ? '1 agent monitored' : 
                 `${agentCount} agents monitored`}
              </p>
            </div>
            <ConnectionStatus 
              isConnected={isConnected} 
              lastUpdate={lastUpdate} 
              agentCount={agentCount}
              onlineCount={onlineCount}
            />
          </div>
        </header>

        {agentCount === 0 ? (
          <div className="text-center text-gray-500 py-16">
            <h2 className="text-2xl font-bold mb-4">No Agents Connected</h2>
            <p className="mb-4">Deploy agents on your servers to start monitoring.</p>
          </div>
        ) : (
          <>
            <div className="mb-6">
              <div className="flex space-x-2 overflow-x-auto pb-2">
                <button
                  onClick={() => setSelectedAgent('overview')}
                  className={`px-4 py-2 rounded-lg whitespace-nowrap transition-colors ${
                    selectedAgent === 'overview' 
                      ? 'bg-blue-600 text-white' 
                      : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                  }`}
                >
                  Overview ({agentCount})
                </button>
                {multiAgentData.summary && multiAgentData.summary.map(agent => (
                  <button
                    key={agent.agent_id}
                    onClick={() => setSelectedAgent(agent.agent_id)}
                    className={`px-4 py-2 rounded-lg whitespace-nowrap transition-colors flex items-center space-x-2 ${
                      selectedAgent === agent.agent_id 
                        ? 'bg-blue-600 text-white' 
                        : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                    }`}
                  >
                    <span>{agent.agent_id}</span>
                    <div className={`w-2 h-2 rounded-full ${agent.is_online ? 'bg-green-400' : 'bg-red-400'}`}></div>
                  </button>
                ))}
              </div>
            </div>

            {selectedAgent === 'overview' ? (
              <>
                <div className="space-y-6">
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                    <StatCard title="Total Agents" value={agentCount}>
                      <div className="text-sm text-gray-400 mt-2">
                        <span className="text-green-400">{onlineCount} online</span> • 
                        <span className="text-red-400 ml-1">{agentCount - onlineCount} offline</span>
                      </div>
                    </StatCard>
                    
                    <StatCard 
                      title="Total Processes" 
                      value={multiAgentData.summary.reduce((sum, a) => sum + (a.process_count || 0), 0)}
                    >
                      <div className="text-sm text-gray-400 mt-2">
                        Across all agents
                      </div>
                    </StatCard>

                    <StatCard 
                      title="Docker Containers" 
                      value={multiAgentData.summary.reduce((sum, a) => sum + (a.docker_count || 0), 0)}
                    >
                      <div className="text-sm text-gray-400 mt-2">
                        Active containers
                      </div>
                    </StatCard>

                    <StatCard 
                      title="Avg CPU Usage" 
                      value={
                        (() => {
                          const avgCpu = multiAgentData.summary
                            .filter(a => a.is_online)
                            .reduce((sum, a) => sum + a.cpu_usage, 0) / Math.max(onlineCount, 1);
                          return `${avgCpu.toFixed(1)}%`;
                        })()
                      }
                    >
                      <div className="text-sm text-gray-400 mt-2">
                        Across {onlineCount} online agents
                      </div>
                    </StatCard>
                  </div>

                  <div>
                    <h2 className="text-2xl font-bold mb-4">All Agents</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      {multiAgentData.summary.map(agent => (
                        <AgentSummaryCard
                          key={agent.agent_id}
                          agent={agent}
                          onClick={setSelectedAgent}
                          isSelected={false}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              </>
            ) : (
              <AgentView 
                agent={multiAgentData.agents[selectedAgent]} 
                formatBytes={formatBytes}
                formatUptime={formatUptime}
              />
            )}
          </>
        )}
      </div>
    </div>
  );
}

export default App;