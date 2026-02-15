import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

interface ProcessingScreenProps {
  setError: (error: string | null) => void;
}

const ProcessingScreen: React.FC<ProcessingScreenProps> = ({ setError }) => {
  const navigate = useNavigate();

  useEffect(() => {
    if (sessionStorage.getItem('initialText')) {
      navigate('/chat')
    }

    const timeoutId = setTimeout(() => {
      setError('Request timed out. Please try again.');
      navigate('/');
    }, 30000);

    return () => clearTimeout(timeoutId);
  }, [navigate, setError]);

  return (
    <div className="flex items-center justify-center min-h-screen bg-white">
      <div className="text-center">
        <div className="animate-spin rounded-full h-10 w-10 border-2 border-gray-200 border-t-blue-600 mx-auto mb-4"></div>
        <div className="text-sm text-gray-500">Starting conversation...</div>
      </div>
    </div>
  );
};

export default ProcessingScreen;
