import { create } from "zustand";

const defaultLanguage = "en-US";

const initialState = {
  role: "",
  topic: "",
  language: defaultLanguage,
  subtitleLanguage: "",
  conversationId: "",
  conversationSecret: "",
  initialAudio: "",
  initialText: "",
  initialSubtitle: "",
  messages: [] as Array<Message>,
  isIntroDone: false,
  hasEnded: false,
};

export interface Message {
  text: string;
  isUser: boolean;
  isAnimated?: boolean;
  hasAnimated?: boolean;
  subtitle?: string;
}

interface ConversationState {
  role: string;
  topic: string;
  language: string;
  subtitleLanguage: string;
  conversationId: string;
  conversationSecret: string;
  initialAudio: string;
  initialText: string;
  initialSubtitle: string;
  messages: Array<Message>;
  isIntroDone: boolean;
  hasEnded: boolean;

  setRole: (role: string) => void;
  setTopic: (topic: string) => void;
  setLanguage: (language: string) => void;
  setSubtitleLanguage: (subtitleLanguage: string) => void;
  setConversationId: (id: string) => void;
  setConversationSecret: (secret: string) => void;
  setInitialAudio: (audio: string) => void;
  setInitialText: (text: string) => void;
  setInitialSubtitle: (subtitle: string) => void;
  setMessages: (messages: Array<Message>) => void;
  addMessage: (message: Message) => void;
  setIsIntroDone: (isIntroDone: boolean) => void;
  setHasEnded: (hasEnded: boolean) => void;

  resetStore: () => void;
}

export const useConversationStore = create<ConversationState>((set) => ({
  ...initialState,

  setRole: (role) => set({ role }),
  setTopic: (topic) => set({ topic }),
  setLanguage: (language) => set({ language }),
  setSubtitleLanguage: (subtitleLanguage) => set({ subtitleLanguage }),
  setConversationId: (id) => set({ conversationId: id }),
  setConversationSecret: (secret) => set({ conversationSecret: secret }),
  setInitialAudio: (audio) => set({ initialAudio: audio }),
  setInitialText: (text) => set({ initialText: text }),
  setInitialSubtitle: (subtitle) => set({ initialSubtitle: subtitle }),
  setMessages: (messages) => set({ messages }),
  addMessage: (message) =>
    set((state) => ({ messages: [...state.messages, message] })),
  setIsIntroDone: (isIntroDone) => set({ isIntroDone }),
  setHasEnded: (hasEnded) => set({ hasEnded }),

  resetStore: () => set({ ...initialState }),
}));
