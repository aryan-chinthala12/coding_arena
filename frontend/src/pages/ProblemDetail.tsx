import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Panel, Group, Separator } from 'react-resizable-panels'
import { ProblemPanel } from '@/components/ProblemPanel'
import { CodeEditorPanel } from '@/components/CodeEditorPanel'
import { ConsolePanel, type ConsoleMessage } from '@/components/ConsolePanel'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import {
  getProblemById,
  getStarterCodeTemplates,
  type Problem,
  type LanguageTemplate,
} from '@/lib/data-service'

// Type definitions for API responses
interface TestCase {
  name: string
  input: string
  expected_output: string
  actual_output: string
}

interface TestResult {
  status: string
  message: string
}

export default function ProblemDetail() {
  const { id: problemId } = useParams<{ id: string }>()
  const navigate = useNavigate()
  
  const [problem, setProblem] = useState<Problem | null>(null)
  const [starterCodeTemplates, setStarterCodeTemplates] = useState<
    LanguageTemplate[]
  >([])
  const [code, setCode] = useState<string>('')
  const [language, setLanguage] = useState<string>('python')
  const [consoleMessages, setConsoleMessages] = useState<ConsoleMessage[]>([])
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false)
  const [isRunning, setIsRunning] = useState<boolean>(false)
  const [isLoading, setIsLoading] = useState<boolean>(true)
  const [error, setError] = useState<string | null>(null)
  const [isConsoleOpen, setIsConsoleOpen] = useState<boolean>(true)
  const [customInput, setCustomInput] = useState<string>('')
  const [showCustomInput, setShowCustomInput] = useState<boolean>(false)
  const [executionStatus, setExecutionStatus] = useState<string>('')
  const [submissionId, setSubmissionId] = useState<string>('')

  // Load problem data and starter code templates
  const loadData = useCallback(async () => {
    if (!problemId) return

    setIsLoading(true)
    setError(null)

    try {
      // Load problem by ID
      const problemData = await getProblemById(problemId)

      if (!problemData) {
        // Problem not found - navigate to 404
        navigate('/not-found')
        return
      }

      setProblem(problemData)

      // Load starter code templates
      const templates = await getStarterCodeTemplates()
      setStarterCodeTemplates(templates)

      // Load initial starter code
      // Default to python if available, otherwise use the first available template
      const defaultTemplate = templates.find((t) => t.id === 'python') || templates[0]
      if (defaultTemplate) {
        setLanguage(defaultTemplate.id)
        setCode(defaultTemplate.starterCode)
      }
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : 'Failed to load problem data'
      setError(errorMessage)
      setConsoleMessages([
        {
          type: 'error',
          message: errorMessage,
          timestamp: Date.now(),
        },
      ])
    } finally {
      setIsLoading(false)
    }
  }, [problemId, navigate])

  useEffect(() => {
    loadData()
  }, [loadData])

  // Handle language change
  const handleLanguageChange = (newLanguage: string) => {
    setLanguage(newLanguage)

    // Load starter code for new language
    const template = starterCodeTemplates.find((t) => t.id === newLanguage)
    if (template) {
      setCode(template.starterCode)
    }
  }

  // Handle code run with sample test cases
  const handleRun = async () => {
    if (!code.trim()) {
      setConsoleMessages([
        {
          type: 'error',
          message: 'Code cannot be empty',
          timestamp: Date.now(),
        },
      ])
      return
    }

    setIsRunning(true)
    setIsConsoleOpen(true)
    setSubmissionId('')
    
    const isCustom = showCustomInput && customInput.trim()
    
    setExecutionStatus(isCustom ? 'Running with custom input...' : 'Running sample test cases...')
    
    setConsoleMessages([])

    try {
      const response = await fetch('/api/run', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          problem_id: problemId,
          language: language,
          source: code,
          custom_input: isCustom ? customInput : null,
        }),
      })

      if (!response.ok) {
        throw new Error(`Run failed: ${response.statusText}`)
      }

      const result = await response.json()

      setExecutionStatus(result.message)

      const messages: ConsoleMessage[] = []

      if (result.test_cases) {
        result.test_cases.forEach((testCase: TestCase) => {
          messages.push({
            type: 'info',
            message: `<div class="space-y-2">
              <div class="font-semibold">${testCase.name}</div>
              <div class="grid grid-cols-1 gap-2 text-sm">
                <div>
                  <div class="font-medium text-gray-700">Input:</div>
                  <pre class="bg-white border border-gray-200 rounded p-2 mt-1 text-xs">${testCase.input}</pre>
                </div>
                <div>
                  <div class="font-medium text-gray-700">Expected Output:</div>
                  <pre class="bg-white border border-gray-200 rounded p-2 mt-1 text-xs">${testCase.expected_output}</pre>
                </div>
                <div>
                  <div class="font-medium text-gray-700">Your Output:</div>
                  <pre class="bg-white border border-gray-200 rounded p-2 mt-1 text-xs">${testCase.actual_output}</pre>
                </div>
              </div>
            </div>`,
            timestamp: Date.now(),
          })
        })
      }

      setConsoleMessages(messages)
    } catch (err) {
      const errorMessage =
        err instanceof Error
          ? err.message
          : 'Run failed: Unable to reach server'
      
      console.error('Run error:', err)
      
      setExecutionStatus('Run failed')
      
      setConsoleMessages([
        {
          type: 'error',
          message: errorMessage,
          timestamp: Date.now(),
        },
      ])
    } finally {
      setIsRunning(false)
    }
  }

  // Handle code submission with all test cases
  const handleSubmit = async () => {
    if (!code.trim()) {
      setConsoleMessages([
        {
          type: 'error',
          message: 'Code cannot be empty',
          timestamp: Date.now(),
        },
      ])
      return
    }

    setIsSubmitting(true)
    setIsConsoleOpen(true)
    setExecutionStatus('Compiling and running all test cases...')
    setSubmissionId('')
    
    setConsoleMessages([])

    try {
      const response = await fetch('/api/submit', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          problem_id: problemId,
          language: language,
          source: code,
        }),
      })

      if (!response.ok) {
        throw new Error(`Submission failed: ${response.statusText}`)
      }

      const result = await response.json()

      setSubmissionId(`Submission ${result.submission_id}`)
      setExecutionStatus(result.message)

      const messages: ConsoleMessage[] = []

      if (result.test_results) {
        result.test_results.forEach((test: TestResult, index: number) => {
          messages.push({
            type: test.status === 'AC' ? 'success' : 'error',
            message: `Test Case ${index + 1}: ${test.status} - ${test.message}`,
            timestamp: Date.now(),
          })
        })
      }

      setConsoleMessages(messages)
    } catch (err) {
      const errorMessage =
        err instanceof Error
          ? err.message
          : 'Submission failed: Unable to reach server'
      
      console.error('Submission error:', err)
      
      setExecutionStatus('Submission failed')
      
      setConsoleMessages([
        {
          type: 'error',
          message: errorMessage,
          timestamp: Date.now(),
        },
      ])
    } finally {
      setIsSubmitting(false)
    }
  }

  // Handle console close
  const handleCloseConsole = () => {
    setIsConsoleOpen(false)
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="h-screen w-screen flex items-center justify-center bg-gray-50 text-gray-800">
        <div className="text-center">
          <div className="flex justify-center mb-4">
            <div className="w-16 h-16 border-4 border-gray-300 border-t-green-600 rounded-full animate-spin"></div>
          </div>
          <div className="text-xl mb-2 font-semibold text-gray-900">Loading problem...</div>
          <div className="text-gray-600 text-sm">Please wait</div>
        </div>
      </div>
    )
  }

  // Error state
  if (error || !problem) {
    return (
      <div className="h-screen w-screen flex items-center justify-center bg-gray-50 text-gray-800 p-4">
        <div className="text-center max-w-md">
          <div className="text-xl mb-2 text-red-600 font-semibold">Error</div>
          <div className="text-gray-600 text-sm mb-4">
            {error || 'Problem not found'}
          </div>
          <div className="flex gap-4 justify-center flex-wrap">
            <button
              onClick={loadData}
              className="px-5 py-2.5 bg-green-600 hover:bg-green-700 text-white font-medium rounded-md transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-green-500"
            >
              Retry
            </button>
            <button
              onClick={() => navigate('/problems')}
              className="px-5 py-2.5 bg-white hover:bg-gray-50 text-gray-900 font-medium rounded-md border border-gray-300 transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-gray-400"
            >
              Back to Problems
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <ErrorBoundary>
      <div className="h-screen w-screen overflow-hidden bg-gray-50">
        <Group orientation="horizontal">
          {/* Left Panel: Problem Description */}
          <Panel defaultSize={40} minSize={25}>
            <ErrorBoundary
              fallback={
                <div className="h-full w-full flex items-center justify-center bg-white text-gray-800">
                  <div className="text-center">
                    <div className="text-xl mb-2 text-red-600 font-semibold">
                      Failed to load problem panel
                    </div>
                    <button
                      onClick={loadData}
                      className="px-5 py-2.5 bg-green-600 hover:bg-green-700 text-white font-medium rounded-md transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-green-500"
                    >
                      Retry
                    </button>
                  </div>
                </div>
              }
            >
              <ProblemPanel problem={problem} />
            </ErrorBoundary>
          </Panel>

          <Separator className="w-2 bg-gray-300 hover:bg-gray-400 transition-colors duration-200 cursor-col-resize active:bg-gray-500" />

          {/* Right Panel: Editor and Console */}
          <Panel defaultSize={60} minSize={35}>
            <Group orientation="vertical">
              {/* Editor Panel */}
              <Panel defaultSize={70} minSize={30}>
                <div className="h-full flex flex-col overflow-hidden">
                  <div className="flex-1 min-h-0">
                    <ErrorBoundary
                    fallback={
                      <div className="h-full w-full flex items-center justify-center bg-white text-gray-800">
                        <div className="text-center">
                          <div className="text-xl mb-2 text-red-600 font-semibold">
                            Failed to load editor
                          </div>
                          <p className="text-gray-600 text-sm mb-4">
                            Using fallback editor
                          </p>
                          <textarea
                            value={code}
                            onChange={(e) => setCode(e.target.value)}
                            className="w-full h-64 bg-gray-50 text-gray-900 font-mono text-sm p-4 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-gray-400 resize-none"
                            spellCheck={false}
                          />
                        </div>
                      </div>
                    }
                  >
                    <CodeEditorPanel
                      value={code}
                      language={language}
                      onChange={setCode}
                      onLanguageChange={handleLanguageChange}
                      availableLanguages={starterCodeTemplates}
                    />
                  </ErrorBoundary>
                  </div>
                  {/* Action Buttons and Custom Input */}
                  <div className="bg-white border-t border-gray-200 shrink-0">
                    <div className="px-4 py-2 flex gap-2 items-center flex-wrap">
                      <button
                        onClick={handleRun}
                        disabled={isSubmitting || isRunning}
                        className="px-4 py-1.5 bg-white hover:bg-gray-50 active:bg-gray-100 disabled:bg-gray-100 disabled:opacity-70 disabled:cursor-not-allowed text-gray-900 text-sm font-medium rounded border border-gray-300 transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-gray-400"
                      >
                        {isRunning ? (
                          <span className="flex items-center gap-1.5">
                            <div className="w-3 h-3 border-2 border-gray-600 border-t-transparent rounded-full animate-spin"></div>
                            Running...
                          </span>
                        ) : (
                          'Run Code'
                        )}
                      </button>
                      <button
                        onClick={handleSubmit}
                        disabled={isSubmitting || isRunning}
                        className="px-4 py-1.5 bg-green-600 hover:bg-green-700 active:bg-green-800 disabled:bg-green-400 disabled:opacity-70 disabled:cursor-not-allowed text-white text-sm font-medium rounded transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-green-500"
                      >
                        {isSubmitting ? (
                          <span className="flex items-center gap-1.5">
                            <div className="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                            Submitting...
                          </span>
                        ) : (
                          'Submit Code'
                        )}
                      </button>
                      <button
                        onClick={() => setShowCustomInput(!showCustomInput)}
                        className="px-3 py-1.5 text-gray-700 hover:text-gray-900 text-sm font-medium transition-colors duration-200"
                      >
                        {showCustomInput ? 'Hide' : 'Custom Input'}
                      </button>
                    </div>
                    {showCustomInput && (
                      <div className="px-4 pb-3">
                        <textarea
                          value={customInput}
                          onChange={(e) => setCustomInput(e.target.value)}
                          placeholder="Enter custom input here..."
                          className="w-full h-20 bg-gray-50 text-gray-900 font-mono text-sm p-2 rounded border border-gray-300 focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-green-500 resize-none"
                          spellCheck={false}
                        />
                      </div>
                    )}
                  </div>
                </div>
              </Panel>

              {isConsoleOpen && (
                <>
                  <Separator className="h-2 bg-gray-300 hover:bg-gray-400 transition-colors duration-200 cursor-row-resize active:bg-gray-500" />
                  
                  {/* Console Panel */}
                  <Panel 
                    defaultSize={30} 
                    minSize={15} 
                  >
                    <ErrorBoundary
                      fallback={
                        <div className="h-full w-full flex items-center justify-center bg-white text-gray-800">
                          <div className="text-center">
                            <div className="text-xl mb-2 text-red-600 font-semibold">
                              Failed to load console
                            </div>
                          </div>
                        </div>
                      }
                    >
                      <ConsolePanel
                        messages={consoleMessages}
                        onClose={handleCloseConsole}
                        status={executionStatus}
                        submissionId={submissionId}
                      />
                    </ErrorBoundary>
                  </Panel>
                </>
              )}
            </Group>
          </Panel>
        </Group>
      </div>
    </ErrorBoundary>
  )
}
