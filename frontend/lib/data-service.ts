/**
 * Data Service for loading problems and starter code templates from JSON files
 * Implements in-memory caching to avoid redundant network requests
 */

// Cache for problems and starter code templates
let problemsCache: Problem[] | null = null
let starterCodeCache: LanguageTemplate[] | null = null

/**
 * Invalidates the problems cache, forcing a fresh fetch on next request
 */
export function invalidateProblemsCache(): void {
  problemsCache = null
}

/**
 * Invalidates the starter code cache, forcing a fresh fetch on next request
 */
export function invalidateStarterCodeCache(): void {
  starterCodeCache = null
}

/**
 * Invalidates all caches
 */
export function invalidateAllCaches(): void {
  problemsCache = null
  starterCodeCache = null
}

export interface Problem {
  id: string
  title: string
  difficulty: 'Easy' | 'Medium' | 'Hard'
  category: string
  points: number
  solvedCount: number
  description: string
  inputFormat: string
  outputFormat: string
  constraints: string[]
  examples: TestExample[]
  timeLimit: number
  memoryLimit: number
}

export interface TestExample {
  input: string
  output: string
  explanation?: string
}

export interface LanguageTemplate {
  id: string
  name: string
  monacoId: string
  extension: string
  starterCode: string
}

interface ProblemsData {
  problems: Problem[]
}

interface StarterCodeData {
  languages: LanguageTemplate[]
}

/**
 * Fetches and parses problems.json with caching and retry logic
 * @param retries - Number of retry attempts (default: 3)
 * @param retryDelay - Delay between retries in milliseconds (default: 1000)
 * @returns Promise resolving to array of problems
 * @throws Error if fetch fails or JSON is invalid after all retries
 */
export async function getProblems(retries: number = 3, retryDelay: number = 1000): Promise<Problem[]> {
  // Return cached data if available
  if (problemsCache !== null) {
    return problemsCache
  }
  
  let lastError: Error | null = null
  
  for (let attempt = 0; attempt <= retries; attempt++) {
    try {
      const response = await fetch('/data/problems.json')
      
      if (!response.ok) {
        throw new Error(`Failed to load problems: ${response.statusText}`)
      }
      
      const data: ProblemsData = await response.json()
      
      if (!data.problems || !Array.isArray(data.problems)) {
        throw new Error('Invalid problems.json structure: missing problems array')
      }
      
      // Cache the result
      problemsCache = data.problems
      
      return data.problems
    } catch (error) {
      lastError = error instanceof Error ? error : new Error('Unknown error')
      
      // If this is not the last attempt, wait before retrying
      if (attempt < retries) {
        await new Promise(resolve => setTimeout(resolve, retryDelay))
      }
    }
  }
  
  // All retries failed
  throw new Error(`Failed to load problems after ${retries + 1} attempts: ${lastError?.message || 'Unknown error'}`)
}

/**
 * Fetches a specific problem by ID from problems.json
 * @param id - The unique problem identifier
 * @returns Promise resolving to the problem or null if not found
 * @throws Error if fetch fails or JSON is invalid
 */
export async function getProblemById(id: string): Promise<Problem | null> {
  try {
    const problems = await getProblems()
    const problem = problems.find(p => p.id === id)
    
    return problem || null
  } catch (error) {
    if (error instanceof Error) {
      throw new Error(`Failed to load problem ${id}: ${error.message}`)
    }
    throw new Error(`Failed to load problem ${id}: Unknown error`)
  }
}

/**
 * Fetches and parses starter-code.json with caching and retry logic
 * @param retries - Number of retry attempts (default: 3)
 * @param retryDelay - Delay between retries in milliseconds (default: 1000)
 * @returns Promise resolving to array of language templates
 * @throws Error if fetch fails or JSON is invalid after all retries
 */
export async function getStarterCodeTemplates(retries: number = 3, retryDelay: number = 1000): Promise<LanguageTemplate[]> {
  // Return cached data if available
  if (starterCodeCache !== null) {
    return starterCodeCache
  }
  
  let lastError: Error | null = null
  
  for (let attempt = 0; attempt <= retries; attempt++) {
    try {
      const response = await fetch('/data/starter-code.json')
      
      if (!response.ok) {
        throw new Error(`Failed to load starter code templates: ${response.statusText}`)
      }
      
      const data: StarterCodeData = await response.json()
      
      if (!data.languages || !Array.isArray(data.languages)) {
        throw new Error('Invalid starter-code.json structure: missing languages array')
      }
      
      // Cache the result
      starterCodeCache = data.languages
      
      return data.languages
    } catch (error) {
      lastError = error instanceof Error ? error : new Error('Unknown error')
      
      // If this is not the last attempt, wait before retrying
      if (attempt < retries) {
        await new Promise(resolve => setTimeout(resolve, retryDelay))
      }
    }
  }
  
  // All retries failed
  throw new Error(`Failed to load starter code templates after ${retries + 1} attempts: ${lastError?.message || 'Unknown error'}`)
}

export interface SubmissionPayload {
  problem_id: string
  language: string
  source: string
}

export interface SubmissionResult {
  submission_id: number
  status: 'AC'
  message: 'Accepted'
  timestamp: number
}

/**
 * Validates and sanitizes a problem ID
 * @param problemId - The problem ID to validate
 * @returns Sanitized problem ID
 * @throws Error if validation fails
 */
function validateProblemId(problemId: string): string {
  if (!problemId || typeof problemId !== 'string') {
    throw new Error('Problem ID must be a non-empty string')
  }
  
  const trimmed = problemId.trim()
  
  if (trimmed.length === 0) {
    throw new Error('Problem ID must be a non-empty string')
  }
  
  // Sanitize: reject IDs with special characters or path traversal attempts
  // eslint-disable-next-line no-control-regex
  if (/[<>:"|?*\x00-\x1f]/.test(trimmed) || trimmed.includes('..')) {
    throw new Error('Problem ID contains invalid characters')
  }
  
  return trimmed
}

/**
 * Validates a programming language
 * @param language - The language to validate
 * @returns Validated language
 * @throws Error if validation fails
 */
function validateLanguage(language: string): string {
  const validLanguages = ['c', 'cpp', 'java', 'python', 'go']
  
  if (!language || typeof language !== 'string') {
    throw new Error('Language must be a non-empty string')
  }
  
  const trimmed = language.trim().toLowerCase()
  
  if (!validLanguages.includes(trimmed)) {
    throw new Error(`Language must be one of: ${validLanguages.join(', ')}`)
  }
  
  return trimmed
}

/**
 * Validates and sanitizes source code
 * @param source - The source code to validate
 * @returns Sanitized source code
 * @throws Error if validation fails
 */
function validateSourceCode(source: string): string {
  if (!source || typeof source !== 'string') {
    throw new Error('Source code must be a non-empty string')
  }
  
  const trimmed = source.trim()
  
  if (trimmed.length === 0) {
    throw new Error('Source code must be a non-empty string')
  }
  
  // Sanitize: remove any null bytes or other control characters that could cause issues
  const sanitized = trimmed.replace(/\0/g, '')
  
  return sanitized
}

/**
 * Submits code to the judge API
 * @param problemId - The problem identifier
 * @param language - The programming language (c, cpp, java, python)
 * @param source - The source code to submit
 * @returns Promise resolving to submission result
 * @throws Error if validation fails or submission fails
 */
export async function submitCode(
  problemId: string,
  language: string,
  source: string
): Promise<SubmissionResult> {
  try {
    // Validate and sanitize all inputs
    const validatedProblemId = validateProblemId(problemId)
    const validatedLanguage = validateLanguage(language)
    const validatedSource = validateSourceCode(source)
    
    // Construct payload with sanitized inputs
    const payload: SubmissionPayload = {
      problem_id: validatedProblemId,
      language: validatedLanguage,
      source: validatedSource
    }
    
    // Send POST request to API
    const response = await fetch('/api/submit', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload)
    })
    
    if (!response.ok) {
      throw new Error(`Submission failed: ${response.statusText}`)
    }
    
    const result: SubmissionResult = await response.json()
    
    return result
  } catch (error) {
    if (error instanceof Error) {
      throw error
    }
    throw new Error('Submission failed: Unknown error')
  }
}
