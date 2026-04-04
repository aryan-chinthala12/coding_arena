# Coding Practice Platform - Frontend

A modern web-based coding practice platform built with React, TypeScript, and Vite. Practice algorithmic problems with an integrated code editor, test runner, and submission system.

## Features

- Browse and filter coding problems by difficulty and category
- Interactive Monaco code editor with syntax highlighting
- Multi-language support (Python, C, C++, Java)
- Resizable panels for optimal workspace layout
- Run code with sample test cases or custom input
- Submit solutions for full test case evaluation
- Real-time console output with test results
- Error boundaries for graceful error handling
- Responsive design with loading states and skeleton screens

## Tech Stack

- **React 19** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **React Router** - Client-side routing
- **Monaco Editor** - Code editor (VS Code's editor)
- **Tailwind CSS** - Styling
- **React Resizable Panels** - Resizable layout

## Getting Started

### Prerequisites

- Node.js 18+ and npm

### Installation

```bash
npm install
```

### Development

```bash
npm run dev
```

The app will be available at `http://localhost:5173`

### Build

```bash
npm run build
```

### Preview Production Build

```bash
npm run preview
```

## Project Structure

```
frontend/
├── components/          # Reusable UI components
│   ├── CodeEditorPanel.tsx
│   ├── ConsolePanel.tsx
│   ├── ErrorBoundary.tsx
│   └── ProblemPanel.tsx
├── lib/                 # Utilities and services
│   ├── data-service.ts  # API and data fetching
│   └── mock-api.ts      # Mock API for development
├── pages/               # Route pages
│   ├── Problems.tsx     # Problems listing
│   ├── ProblemDetail.tsx # Problem solver view
│   └── NotFound.tsx     # 404 page
├── public/
│   └── data/            # Static JSON data
│       ├── problems.json
│       └── starter-code.json
└── src/
    ├── App.tsx          # Root component with routing
    ├── main.tsx         # Entry point
    └── index.css        # Global styles
```

## Available Routes

- `/` - Redirects to problems list
- `/problems` - Browse all problems
- `/problem/:id` - Solve a specific problem
- `/not-found` - 404 page

## API Integration

The frontend expects a judge server API with the following endpoints:

- `POST /api/run` - Run code with sample test cases
- `POST /api/submit` - Submit code for evaluation

See `lib/data-service.ts` for payload structures.

## Code Quality

### Linting

```bash
npm run lint
```

### Type Checking

TypeScript is configured with strict mode. Run type checking with:

```bash
npx tsc --noEmit
```
