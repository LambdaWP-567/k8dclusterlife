import './index.css'

function App() {
  return (
    <div className="min-h-screen bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100">
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <h1 className="text-4xl font-bold mb-4">k8dclusterlife</h1>
          <p className="text-gray-500 dark:text-gray-400">Kubernetes Monitoring + KI-Selbstheilung</p>
          <div className="mt-8 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
            <p className="text-green-700 dark:text-green-400 font-medium">✓ Alle Cluster gesund</p>
          </div>
        </div>
      </div>
    </div>
  )
}

export default App
