import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import AnimatedText from './AnimatedText';
import Navbar from './Navbar';
import { Message, useConversationStore } from '../store';

const convertToWav = async (blob: Blob): Promise<Blob> => {
  const audioContext = new AudioContext();
  const arrayBuffer = await blob.arrayBuffer();
  const audioBuffer = await audioContext.decodeAudioData(arrayBuffer);

  const numChannels = 1;
  const sampleRate = audioBuffer.sampleRate;
  const samples = audioBuffer.getChannelData(0);
  const buffer = new ArrayBuffer(44 + samples.length * 2);
  const view = new DataView(buffer);

  // WAV header
  const writeString = (offset: number, str: string) => {
    for (let i = 0; i < str.length; i++) view.setUint8(offset + i, str.charCodeAt(i));
  };
  writeString(0, 'RIFF');
  view.setUint32(4, 36 + samples.length * 2, true);
  writeString(8, 'WAVE');
  writeString(12, 'fmt ');
  view.setUint32(16, 16, true);
  view.setUint16(20, 1, true);
  view.setUint16(22, numChannels, true);
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * numChannels * 2, true);
  view.setUint16(32, numChannels * 2, true);
  view.setUint16(34, 16, true);
  writeString(36, 'data');
  view.setUint32(40, samples.length * 2, true);

  // PCM samples
  let offset = 44;
  for (let i = 0; i < samples.length; i++, offset += 2) {
    const s = Math.max(-1, Math.min(1, samples[i]));
    view.setInt16(offset, s < 0 ? s * 0x8000 : s * 0x7FFF, true);
  }

  await audioContext.close();
  return new Blob([buffer], { type: 'audio/wav' });
};

interface ChatScreenProps {
  backendHost: string;
  setError: (error: string | null) => void;
}

const ChatScreen: React.FC<ChatScreenProps> = ({ backendHost, setError }) => {
  const { messages, initialText, initialAudio, language, subtitleLanguage, isIntroDone, conversationId, conversationSecret, hasEnded, addMessage, setIsIntroDone, setHasEnded, resetStore } = useConversationStore();

  const [isRecording, setIsRecording] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const [hasStarted, setHasStarted] = useState(false);
  const hasSubtitles = subtitleLanguage !== '';
  const [showSubtitles, setShowSubtitles] = useState(hasSubtitles);

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
        synthesizeText(initialText, language);
      }
      setIsIntroDone(true);
    }
  }, [isIntroDone, initialText, initialAudio, language, setIsIntroDone])

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

      mediaRecorderRef.current.onstop = async () => {
        const webmBlob = new Blob(audioChunks, { type: 'audio/webm' });
        const wavBlob = await convertToWav(webmBlob);
        sendAudioToServer(wavBlob);
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
    formData.append('file', audioBlob, 'audio.wav');

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
          synthesizeText(data?.data?.answer?.text, data?.data?.language);
        }

        setHasStarted(true);

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

  const synthesizeText = async (text: string, language: string) => {
    if (!text) {
      return
    }

    if (window.speechSynthesis.speaking) {
      window.speechSynthesis.cancel()
    }

    const audio = new SpeechSynthesisUtterance();
    audio.text = text;
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
          synthesizeText(data?.data?.answer?.text, data?.data?.language);
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
    <div className="flex flex-col h-screen bg-white">
      <Navbar
        backendHost={backendHost}
        showBackIcon
        showForwardIcon
        showStartOver
        onBack={handleBack}
        onStartOver={handleStartOver}
        disableForward={true}
      />
      <div ref={chatContainerRef} className="flex-grow overflow-y-auto px-4 py-4">
        <div className="max-w-2xl mx-auto space-y-3">
          {messages.map((message, index) => (
            <div key={index} className={`flex ${message.isUser ? 'justify-end' : 'justify-start'}`}>
              <div className="max-w-[80%]">
                <div className={`inline-block px-4 py-2.5 rounded-2xl text-sm leading-relaxed ${message.isUser
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-900'
                  }`}>
                  {message.isAnimated
                    ? <AnimatedText message={message} />
                    : message.text
                  }
                </div>
                {showSubtitles && message.subtitle && (
                  <div className={`mt-1 text-xs text-gray-400 italic px-1 ${message.isUser ? 'text-right' : 'text-left'}`}>
                    {message.subtitle}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="border-t border-gray-100 bg-white px-4 py-3">
        <div className="max-w-2xl mx-auto flex gap-3 items-center">
          <button
            onClick={() => setShowSubtitles(!showSubtitles)}
            disabled={!hasSubtitles}
            className={`p-3 rounded-xl transition-all ${!hasSubtitles
              ? 'text-gray-300 cursor-not-allowed'
              : showSubtitles
                ? 'text-blue-600 bg-blue-50'
                : 'text-gray-400 hover:text-gray-600 hover:bg-gray-50'
              }`}
            title={!hasSubtitles ? 'No subtitle language selected' : showSubtitles ? 'Hide subtitles' : 'Show subtitles'}
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
            </svg>
          </button>
          <button
            onClick={isRecording ? stopRecording : startRecording}
            disabled={isProcessing || hasEnded}
            className={`flex-grow py-3 rounded-xl font-medium text-sm transition-all ${isProcessing || hasEnded
              ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
              : isRecording
                ? 'bg-red-500 text-white animate-pulse'
                : 'bg-blue-600 text-white hover:bg-blue-700'
              }`}
          >
            {isProcessing
              ? 'Processing...'
              : isRecording
                ? 'Stop Recording'
                : hasEnded
                  ? 'Conversation ended'
                  : 'Record Answer'
            }
          </button>
          {hasStarted && !hasEnded && (
            <button
              onClick={endConversation}
              disabled={isProcessing || isRecording || hasEnded}
              className={`px-5 py-3 rounded-xl font-medium text-sm transition-all ${isProcessing || isRecording || hasEnded
                ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                : 'bg-white border border-red-300 text-red-500 hover:bg-red-50'
                }`}
            >
              End
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default ChatScreen;
