// home/web/src/App.jsx
import React, { useState, useEffect } from 'react';

// A simple card component for displaying stats
const StatCard = ({ title, value, children }) => (
  <div className="bg-gray-800 p-6 rounded-lg shadow-lg">
    <h3 className="text-lg font-semibold text-gray-400">{title}</h3>
    <p className="text-3xl font-bold text-white mt-2">{value}</p>
    {children}
  </div>
);

// A progress bar component
const ProgressBar = ({ value, max }) => {
    const percentage = (value / max) * 100;
    return (
        <div className="w-full bg-gray-700 rounded-full h-4 mt-2">
            <div
                className="bg-blue-500 h-4 rounded-full"
                style={{ width: `${percentage}%` }}
            ></div>
        </div>
    );
};


function App() {
  const [metrics, setMetrics] = useState(null);
  const [socket, setSocket] = useState(null);

  useEffect(() => {
    // Determine WebSocket protocol based on window location
    const wsProtocol = window.location.protocol === 'https' ? 'wss' : 'ws';
    const wsUrl = `${wsProtocol}://${window.location.host}/ws`;

    const newSocket = new WebSocket(wsUrl);
    setSocket(newSocket);

    newSocket.onopen = () => {
      console.log('WebSocket connected');
    };

    newSocket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      setMetrics(data);
    };

    newSocket.onclose = () => {
      console.log('WebSocket disconnected');
    };

    // Cleanup on component unmount
    return () => newSocket.close();
  }, []);

  if (!metrics) {
    return (
      <div className="min-h-screen bg-gray-900 text-white flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-4xl font-bold">Connecting to VPS...</h1>
          <p className="mt-2 text-gray-400">Waiting for the first metrics from the agent.</p>
        </div>
      </div>
    );
  }
  
  // Helper to format bytes into GB
  const formatBytes = (bytes) => (bytes / (1024 * 1024 * 1024)).toFixed(2);


  return (
    <div className="min-h-screen bg-gray-900 text-white p-8">
      <div className="max-w-7xl mx-auto">
        <header className="mb-8">
          <h1 className="text-4xl font-bold">VPS Monitor</h1>
          <p className="text-gray-400">
            Live metrics for <span className="font-semibold text-blue-400">{metrics.hostname}</span>
          </p>
        </header>

        <main className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <StatCard title="CPU Usage" value={`${metrics.cpu_usage.toFixed(2)}%`}>
            <ProgressBar value={metrics.cpu_usage} max={100} />
          </StatCard>
          
          <StatCard title="Memory" value={`${(metrics.memory.usedPercent).toFixed(2)}%`}>
            <ProgressBar value={metrics.memory.used} max={metrics.memory.total} />
            <div className="text-sm text-gray-400 mt-2">
                {formatBytes(metrics.memory.used)} GB / {formatBytes(metrics.memory.total)} GB
            </div>
          </StatCard>

          <StatCard title="Disk Space" value={`${(metrics.disk.usedPercent).toFixed(2)}%`}>
             <ProgressBar value={metrics.disk.used} max={metrics.disk.total} />
             <div className="text-sm text-gray-400 mt-2">
                {formatBytes(metrics.disk.used)} GB / {formatBytes(metrics.disk.total)} GB
            </div>
          </StatCard>
          
          {/* Process List */}
          <div className="md:col-span-2 lg:col-span-3 bg-gray-800 p-6 rounded-lg shadow-lg">
            <h3 className="text-lg font-semibold text-gray-400 mb-4">Running Processes</h3>
            <div className="overflow-x-auto">
                <table className="w-full text-left">
                    <thead>
                        <tr className="border-b border-gray-700">
                            <th className="p-2">PID</th>
                            <th className="p-2">Name</th>
                            <th className="p-2">CPU %</th>
                            <th className="p-2">Mem %</th>
                        </tr>
                    </thead>
                    <tbody>
                        {metrics.processes && metrics.processes.map(p => (
                            <tr key={p.pid} className="border-b border-gray-700 hover:bg-gray-700">
                                <td className="p-2">{p.pid}</td>
                                <td className="p-2 font-mono">{p.name}</td>
                                <td className="p-2">{p.cpu_percent.toFixed(2)}</td>
                                <td className="p-2">{p.memory_percent.toFixed(2)}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}

export default App;
