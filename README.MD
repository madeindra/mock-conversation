# Mock Conversation

An AI-powered mock conversation tool that lets you practice conversations in any language, on any topic, with any AI role. Built with a Go backend and React TypeScript frontend.

## Features

- **Any Role**: Choose what role the AI plays (e.g., Spanish tutor, debate partner, travel guide, doctor)
- **Any Topic**: Set any conversation topic
- **Any Language**: 15+ supported languages for conversation
- **Subtitles**: Optional translation subtitles in a different language, toggleable during conversation
- **Voice Interaction**: Speak and listen with audio recording and text-to-speech
- **Random Voice**: Each conversation gets a unique OpenAI TTS voice, consistent throughout the session
- **Natural Flow**: AI detects when the conversation is ending and responds naturally
- **Structured JSON Responses**: Single ChatGPT API call per interaction returns transcript, response, subtitles, and conversation state

## Architecture

All powered by OpenAI APIs:

1. **ChatGPT** — generates all text content (transcript, response, subtitles, end-of-conversation flag) in a single structured JSON response
2. **OpenAI TTS** — generates audio from the response text

### Models

- `gpt-4o-mini-transcribe` — audio transcription (answer chat)
- `gpt-4o-mini` — text chat with JSON response format (start chat, answer chat, end chat)
- `gpt-4o-mini-tts` — text-to-speech

## Server

The server is built with Go and is located in the `server` directory. It requires the following environment variables:

- `OPENAI_API_KEY`: Your OpenAI API key (required)
- `DB_PATH`: Path to SQLite database (optional, defaults to `./app.db`)

Optional configurations:
- `PORT`: The port number for the server to run (defaults to 8080)
- `CORS_ALLOWED_ORIGINS`: Allowed origin to call the APIs
- `CORS_ALLOWED_METHODS`: Allowed methods of the APIs call
- `CORS_ALLOWED_HEADERS`: Allowed headers of the APIs call

## Client

The client is built using React TypeScript with Vite and Node.js 20. It is located in the `client` directory.

- `VITE_BACKEND_URL`: URL of the backend server (optional, defaults to `http://0.0.0.0:8080`)

## Running with Docker

1. Make sure you have Docker and docker-compose installed.

2. Create a `.env` file in the root directory:

   ```
   OPENAI_API_KEY=your_openai_api_key_here
   ```

3. Run:

   ```
   docker-compose up --build
   ```

4. Open `http://localhost:3000` in your browser.

## Development

### Server

```bash
cd server
go run main.go
```

### Client

```bash
cd client
npm install
npm run dev
```

### Both (using Makefile)

```bash
make run
```
