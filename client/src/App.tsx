import React, { useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import StartScreen from './components/StartScreen';
import ProcessingScreen from './components/ProcessingScreen';
import ChatScreen from './components/ChatScreen';

const App: React.FC = () => {
  const backendHost = import.meta.env.VITE_BACKEND_URL || 'http://0.0.0.0:8080';
  const [error, setError] = useState<string | null>(null);

  return (
    <Router>
      <div className="min-h-screen bg-white text-gray-900 flex flex-col relative">
        {error && (
          <div className="absolute top-4 left-4 right-4 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg z-50 flex items-center justify-between" role="alert">
            <span className="text-sm">{error}</span>
            <button onClick={() => setError(null)} className="text-red-400 hover:text-red-600 ml-4">
              <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                <path d="M14.348 14.849a1.2 1.2 0 0 1-1.697 0L10 11.819l-2.651 3.029a1.2 1.2 0 1 1-1.697-1.697l2.758-3.15-2.759-3.152a1.2 1.2 0 1 1 1.697-1.697L10 8.183l2.651-3.031a1.2 1.2 0 1 1 1.697 1.697l-2.758 3.152 2.758 3.15a1.2 1.2 0 0 1 0 1.698z"/>
              </svg>
            </button>
          </div>
        )}
        <div className="flex-grow">
          <Routes>
            <Route path="/" element={<StartScreen backendHost={backendHost} setError={setError} />} />
            <Route path="/processing" element={<ProcessingScreen setError={setError} />} />
            <Route path="/chat" element={<ChatScreen backendHost={backendHost} setError={setError} />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </div>
      </div>
    </Router>
  );
};

export default App;
