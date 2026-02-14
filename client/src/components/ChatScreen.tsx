import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import AnimatedText from './AnimatedText';
import Navbar from './Navbar';
import { Message, useConversationStore } from '../store';

interface ChatScreenProps {
  backendHost: string;
  setError: (error: string | null) => void;
}

const ChatScreen: React.FC<ChatScreenProps> = ({ backendHost, setError }) => {
  const { messages, initialText, initialSSML, initialAudio, language, isIntroDone, conversationId, conversationSecret, hasEnded, addMessage, setIsIntroDone, setHasEnded, resetStore } = useConversationStore();

  const [isRecording, setIsRecording] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const [hasStarted, setHasStarted] = useState(false);

  const navigate = useNavigate();

  const audioRef = useRef<HTMLAudioElement | null>(null);
  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const chatContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!initialText) {
      navigate('/');
    }
  }, [navigate, initialText]);

  useEffect(() => {
    if (!isIntroDone) {
      if (initialAudio && initialAudio !== 'undefined') {
        playAudio(initialAudio);
      } else {
        synthesizeText(initialText, initialSSML, language);
      }
      setIsIntroDone(true);
    }
  }, [isIntroDone, initialText, initialSSML, initialAudio, language, setIsIntroDone])

  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, [messages]);

  const startRecording = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      mediaRecorderRef.current = new MediaRecorder(stream);

      const audioChunks: BlobPart[] = [];
      mediaRecorderRef.current.ondataavailable = (event) => {
        audioChunks.push(event.data);
      };

      mediaRecorderRef.current.onstop = () => {
        const audioBlob = new Blob(audioChunks, { type: 'audio/webm' });
        sendAudioToServer(audioBlob);
      };

      mediaRecorderRef.current.start();
      setIsRecording(true);
    } catch (error) {
      console.error('Error accessing microphone:', error);
      setError('Failed to access microphone. Please check your permissions and try again.');
    }
  };

  const stopRecording = () => {
    if (mediaRecorderRef.current && isRecording) {
      mediaRecorderRef.current.stop();
      setIsRecording(false);
    }
  };

  const sendAudioToServer = async (audioBlob: Blob) => {
    const formData = new FormData();
    formData.append('file', audioBlob, 'audio.webm');

    const authString = btoa(`${conversationId}:${conversationSecret}`);

    setIsProcessing(true);

    try {
      const response = await fetch(`${backendHost}/chat/answer`, {
        method: 'POST',
        headers: {
          'Authorization': `Basic ${authString}`,
        },
        body: formData,
      });

      const data = await response.json();

      if (response.ok && data.data) {
        const userMessage: Message = {
          text: data.data.prompt.text,
          isUser: true,
          subtitle: data.data.prompt.subtitle || '',
        };
        const botMessage: Message = {
          text: data.data.answer.text,
          isUser: false,
          isAnimated: true,
          subtitle: data.data.answer.subtitle || '',
        };

        addMessage(userMessage);
        addMessage(botMessage);

        if (data?.data?.answer?.audio) {
          playAudio(data.data.answer.audio);
        } else {
          synthesizeText(data?.data?.answer?.text, data?.data?.answer?.ssml, data?.data?.language);
        }

        setHasStarted(true);

        // Auto-end conversation if AI detected user wants to end
        if (data.data.isLast) {
          setHasEnded(true);
        }
      } else {
        const errorMessage = data.message || 'Failed to process your response. Please try again.';
        setError(errorMessage);
      }
    } catch (error) {
      console.error('Error sending audio:', error);
      setError('Failed to send your response. Please check your connection and try again.');
    } finally {
      setIsProcessing(false);
    }
  };

  const playAudio = (base64Audio: string | null) => {
    if (!base64Audio) {
      return
    }

    stopAudio();

    audioRef.current = new Audio(`data:audio/mp3;base64,${base64Audio}`);
    audioRef.current.play();
  };

  const stopAudio = () => {
    if (audioRef.current) {
      audioRef.current.pause();
      audioRef.current.currentTime = 0;
    }
  };

  const synthesizeText = async (text: string, ssml: string, language: string) => {
    if (!text && !ssml) {
      return
    }

    if (window.speechSynthesis.speaking) {
      window.speechSynthesis.cancel()
    }

    const audio = new SpeechSynthesisUtterance();
    audio.text = ssml || text;
    audio.lang = language;
    audio.rate = 1.2;
    window.speechSynthesis.speak(audio);
  }

  const endConversation = async () => {
    const authString = btoa(`${conversationId}:${conversationSecret}`);

    setIsProcessing(true);

    try {
      const response = await fetch(`${backendHost}/chat/end`, {
        method: 'GET',
        headers: {
          'Authorization': `Basic ${authString}`,
        },
      });

      const data = await response.json();

      if (response.ok && data.data) {
        const botMessage: Message = {
          text: data.data.answer.text,
          isUser: false,
          isAnimated: true,
          subtitle: data.data.answer.subtitle || '',
        };
        addMessage(botMessage);

        if (data?.data?.answer?.audio) {
          playAudio(data.data.answer.audio);
        } else {
          synthesizeText(data?.data?.answer?.text, data?.data?.answer?.ssml, data?.data?.language);
        }
        setHasEnded(true);
      } else {
        setError(data.message || 'Failed to end the conversation. Please try again.');
      }
    } catch (error) {
      console.error('Error ending conversation:', error);
      setError('Failed to end the conversation. Please check your connection and try again.');
    } finally {
      setIsProcessing(false);
    }
  };

  const handleStartOver = () => {
    stopAudio();

    resetStore();
    navigate('/');
  };

  const handleBack = () => {
    stopAudio();

    navigate('/');
  };

  return (
    <div className="flex flex-col h-screen bg-[#1E1E2E] text-white">
      <Navbar
        backendHost={backendHost}
        showBackIcon
        showForwardIcon
        showStartOver
        onBack={handleBack}
        onStartOver={handleStartOver}
        disableForward={true}
      />
      <div ref={chatContainerRef} className="flex-grow overflow-y-auto px-4 py-2">
        {messages.map((message, index) => (
          <div key={index} className={`mb-4 ${message.isUser ? 'text-right' : 'text-left'}`}>
            <span className={`inline-block p-3 rounded-2xl ${message.isUser
              ? 'bg-[#3E64FF] text-white'
              : 'bg-[#2B2B3B] text-white'
              }`}>
              {message.isAnimated
                ? <AnimatedText message={message} />
                : message.text
              }
            </span>
            {message.subtitle && (
              <div className={`mt-1 text-sm text-gray-400 italic ${message.isUser ? 'text-right' : 'text-left'}`}>
                {message.subtitle}
              </div>
            )}
          </div>
        ))}
      </div>

      <div className="flex justify-between items-center space-x-4 p-4 bg-[#1E1E2E]">
        <button
          onClick={isRecording ? stopRecording : startRecording}
          disabled={isProcessing || hasEnded}
          className={`w-full p-4 rounded-xl font-bold text-lg transition-all duration-300 ${isProcessing || hasEnded
            ? 'bg-[#2B2B3B] text-gray-400 cursor-not-allowed'
            : isRecording
              ? 'bg-[#FF3E3E] text-white animate-pulse'
              : 'bg-[#3E64FF] text-white hover:bg-opacity-90'
            }`}
        >
          {isProcessing
            ? 'Processing...'
            : isRecording
              ? 'Stop Recording'
              : hasEnded
                ? 'This conversation has ended'
                : 'Record Answer'
          }
        </button>
        {hasStarted && !hasEnded && (
          <button
            onClick={endConversation}
            disabled={isProcessing || isRecording || hasEnded}
            className={`w-3/12 p-4 rounded-xl font-bold text-lg hover:bg-opacity-90 transition-all duration-300 ${isProcessing || isRecording || hasEnded
              ? 'bg-[#2B2B3B] text-gray-400 cursor-not-allowed'
              : 'bg-[#FF3E3E] text-white hover:bg-opacity-90'
              }`}
          >
            End
          </button>
        )}
      </div>
    </div>
  );
};

export default ChatScreen;
