import React from 'react';
import { useNavigate } from 'react-router-dom';
import Navbar from './Navbar';
import { useConversationStore } from '../store';

interface StartScreenProps {
  backendHost: string;
  setError: (error: string | null) => void;
}

const languageOptions = [
  { name: "English", code: "en-US" },
  { name: "Bahasa Indonesia", code: "id-ID" },
  { name: "Spanish", code: "es-ES" },
  { name: "French", code: "fr-FR" },
  { name: "German", code: "de-DE" },
  { name: "Portuguese", code: "pt-BR" },
  { name: "Italian", code: "it-IT" },
  { name: "Japanese", code: "ja-JP" },
  { name: "Korean", code: "ko-KR" },
  { name: "Chinese (Mandarin)", code: "zh-CN" },
  { name: "Arabic", code: "ar-SA" },
  { name: "Hindi", code: "hi-IN" },
  { name: "Russian", code: "ru-RU" },
  { name: "Dutch", code: "nl-NL" },
  { name: "Turkish", code: "tr-TR" },
];

const StartScreen: React.FC<StartScreenProps> = ({ backendHost, setError }) => {
  const {
    role, topic, language, subtitleLanguage, messages,
    setIsIntroDone, setMessages, setRole, setTopic, setLanguage,
    setSubtitleLanguage, setConversationId, setConversationSecret,
    setInitialAudio, setInitialText, setInitialSSML, setInitialSubtitle,
    setHasEnded,
  } = useConversationStore();

  const navigate = useNavigate();

  // Filter out selected conversation language from subtitle options
  const subtitleOptions = languageOptions.filter(lang => lang.code !== language);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    setMessages([]);
    setHasEnded(false);

    navigate('/processing');

    try {
      const response = await fetch(`${backendHost}/chat/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ role, topic, language, subtitleLanguage }),
      });

      const data = await response.json();

      if (response.ok && data.data) {
        setConversationId(data.data?.id);
        setConversationSecret(data.data?.secret);
        setInitialAudio(data.data?.audio);
        setInitialText(data.data?.text);
        setInitialSSML(data.data?.ssml);
        setInitialSubtitle(data.data?.subtitle || '');
        setLanguage(data.data?.language);

        setMessages([{
          text: data.data?.text,
          isUser: false,
          isAnimated: true,
          subtitle: data.data?.subtitle || '',
        }]);
        setIsIntroDone(false);

        navigate('/chat');
      } else {
        const errorMessage = data.message || 'Failed processing your request, please try again';
        setError(errorMessage);
        navigate('/');
      }
    } catch (error) {
      console.error('Error starting conversation:', error);
      setError('Failed processing your request, please try again');
      navigate('/');
    }
  };

  const handleForward = () => {
    navigate('/chat');
  };

  return (
    <div className="flex flex-col h-screen bg-[#1E1E2E] text-white">
      {messages.length > 0 && (
        <Navbar
          backendHost={backendHost}
          showBackIcon
          showForwardIcon
          onForward={handleForward}
          disableBack={true}
        />
      )}
      <div className="container mx-auto mt-10 p-4 flex-grow">
        <div className="max-w-md mx-auto bg-[#2B2B3B] p-8 rounded-xl shadow-lg">
          <h1 className="text-3xl font-bold mb-6 text-center text-white">Mock Conversation</h1>
          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label htmlFor="role" className="block mb-2 text-white font-semibold">AI Role</label>
              <input
                type="text"
                id="role"
                value={role}
                onChange={(e) => setRole(e.target.value)}
                placeholder="e.g. Spanish tutor, debate partner, travel guide"
                className="w-full p-3 bg-[#3A3A4E] text-white border border-[#4A4A5E] rounded-lg focus:outline-none focus:ring-2 focus:ring-[#3E64FF]"
                required
              />
            </div>
            <div>
              <label htmlFor="topic" className="block mb-2 text-white font-semibold">Topic</label>
              <input
                type="text"
                id="topic"
                value={topic}
                onChange={(e) => setTopic(e.target.value)}
                placeholder="e.g. Traveling in Japan, learning to cook, philosophy"
                className="w-full p-3 bg-[#3A3A4E] text-white border border-[#4A4A5E] rounded-lg focus:outline-none focus:ring-2 focus:ring-[#3E64FF]"
                required
              />
            </div>
            <div>
              <label htmlFor="language" className="block mb-2 text-white font-semibold">Conversation Language</label>
              <select
                id="language"
                value={language}
                onChange={(e) => {
                  setLanguage(e.target.value);
                  // Reset subtitle if it matches new language
                  if (subtitleLanguage === e.target.value) {
                    setSubtitleLanguage('');
                  }
                }}
                className="w-full p-3 bg-[#3A3A4E] text-white border border-[#4A4A5E] rounded-lg focus:outline-none focus:ring-2 focus:ring-[#3E64FF]"
                required
              >
                {languageOptions.map((lang) => (
                  <option key={lang.code} value={lang.code}>{lang.name}</option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor="subtitleLanguage" className="block mb-2 text-white font-semibold">Subtitle Language (Optional)</label>
              <select
                id="subtitleLanguage"
                value={subtitleLanguage}
                onChange={(e) => setSubtitleLanguage(e.target.value)}
                className="w-full p-3 bg-[#3A3A4E] text-white border border-[#4A4A5E] rounded-lg focus:outline-none focus:ring-2 focus:ring-[#3E64FF]"
              >
                <option value="">No subtitles</option>
                {subtitleOptions.map((lang) => (
                  <option key={lang.code} value={lang.code}>{lang.name}</option>
                ))}
              </select>
            </div>
            <button type="submit" className="w-full p-4 bg-[#3E64FF] text-white font-bold rounded-xl hover:bg-opacity-90 transition-all duration-300">
              Start Conversation
            </button>
          </form>
        </div>
      </div>
    </div>
  );
};

export default StartScreen;
