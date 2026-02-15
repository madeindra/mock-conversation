import React, { useState, useEffect } from 'react';

interface NavbarProps {
  backendHost: string;
  showBackIcon?: boolean;
  showForwardIcon?: boolean;
  showStartOver?: boolean;
  onBack?: () => void;
  onForward?: () => void;
  onStartOver?: () => void;
  disableBack?: boolean;
  disableForward?: boolean;
}

interface StatusResponse {
  message: string;
  data: {
    backend: boolean;
    api: boolean | null;
    apiStatus: string | null;
    key: boolean;
  };
}

const Navbar: React.FC<NavbarProps> = ({
  backendHost,
  showBackIcon = false,
  showForwardIcon = false,
  showStartOver = false,
  onBack,
  onForward,
  onStartOver,
  disableBack = false,
  disableForward = false
}) => {
  const [status, setStatus] = useState<StatusResponse['data'] | null>(null);
  const [showTooltip, setShowTooltip] = useState(false);

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await fetch(`${backendHost}/chat/status`);
        if (!response.ok) {
          throw new Error('Network response was not ok');
        }
        const data: StatusResponse = await response.json();
        setStatus(data.data);
      } catch (error) {
        console.error('Error fetching status:', error);
        setStatus(null);
      }
    };

    fetchStatus();
    const intervalId = setInterval(fetchStatus, 30000);

    return () => clearInterval(intervalId);
  }, [backendHost]);

  const getStatusColor = () => {
    if (!status) return 'bg-red-400';
    if (status.backend && status.api === true && status.key) return 'bg-emerald-400';
    if (status.api === false) return 'bg-amber-400';
    return 'bg-red-400';
  };

  const capitalizeFirstLetter = (string: string) => {
    return string.charAt(0).toUpperCase() + string.slice(1);
  };

  if (!showBackIcon && !showForwardIcon && !showStartOver && !status) {
    return null;
  }

  const handleBack = () => {
    if (!disableBack && onBack) {
      onBack();
    }
  }

  const handleForward = () => {
    if (!disableForward && onForward) {
      onForward();
    }
  }

  const handleStartOver = () => {
    if (onStartOver) {
      const isConfirmed = window.confirm("Are you sure you want to end this conversation and start over?");
      if (isConfirmed) {
        onStartOver();
      }
    }
  };

  return (
    <nav className="px-4 py-3 flex justify-between items-center border-b border-gray-100 bg-white relative">
      <div className="flex items-center gap-2">
        {showBackIcon && (
          <button
            onClick={handleBack}
            className={`p-1.5 rounded-md transition-colors ${disableBack ? 'text-gray-300 cursor-not-allowed' : 'text-gray-500 hover:text-gray-900 hover:bg-gray-100'}`}
            disabled={disableBack}
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
        )}
        {showForwardIcon && (
          <button
            onClick={handleForward}
            className={`p-1.5 rounded-md transition-colors ${disableForward ? 'text-gray-300 cursor-not-allowed' : 'text-gray-500 hover:text-gray-900 hover:bg-gray-100'}`}
            disabled={disableForward}
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </button>
        )}
      </div>
      <div className="flex items-center gap-3">
        <div
          className="relative"
          onMouseEnter={() => setShowTooltip(true)}
          onMouseLeave={() => setShowTooltip(false)}
        >
          <div className={`w-2.5 h-2.5 rounded-full ${getStatusColor()}`} />
          {showTooltip && (
            <div className="absolute top-full right-0 mt-2 p-3 bg-white border border-gray-200 rounded-lg shadow-lg z-10 text-xs text-gray-600 whitespace-nowrap">
              <p>Server: {status?.backend ? 'Up' : 'Down'}</p>
              <p>API: {status?.api === null ? 'Unknown' : (status?.api ? 'Up' : 'Down')}</p>
              <p>Status: {status?.apiStatus ? capitalizeFirstLetter(status?.apiStatus) : 'Unknown'}</p>
              <p>Authorized: {status?.key ? 'Yes' : 'No'}</p>
            </div>
          )}
        </div>
        {showStartOver && (
          <button
            onClick={handleStartOver}
            className="p-1.5 rounded-md text-gray-400 hover:text-gray-900 hover:bg-gray-100 transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        )}
      </div>
    </nav>
  );
};

export default Navbar;
