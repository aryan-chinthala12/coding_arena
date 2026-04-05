/**
 * Mock API interceptor for Vite
 * This simulates backend API responses by intercepting fetch calls
 */

interface RunRequestBody {
  problem_id?: string;
  language?: string;
  source?: string;
  custom_input?: string | null;
}

if (typeof window !== 'undefined') {
  const originalFetch = window.fetch
  window.fetch = async (...args) => {
    const [url, options] = args
    const urlString = typeof url === 'string' ? url : url instanceof URL ? url.toString() : ''

    if (urlString.startsWith('/api/submit') && (!options || options.method === 'POST')) {
      console.log('Mocking API call to /api/submit')
      
      try {
        await new Promise(resolve => setTimeout(resolve, 800))
        
        const testResults = [
          { status: 'AC', message: 'Accepted' },
          { status: 'AC', message: 'Accepted' },
          { status: 'WA', message: 'Wrong Answer' },
          { status: 'AC', message: 'Accepted' },
          { status: 'TLE', message: 'Time Limit Exceeded' },
        ]
        
        const passedCount = testResults.filter(t => t.status === 'AC').length
        const totalCount = testResults.length
        const allPassed = passedCount === totalCount
        
        const responseData = {
          submission_id: Date.now(),
          status: allPassed ? 'AC' : 'WA',
          message: allPassed ? 'All test cases passed' : `${passedCount}/${totalCount} test cases passed`,
          test_results: testResults,
          timestamp: Date.now()
        }
        
        return new Response(JSON.stringify(responseData), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        })
      } catch {
        return new Response(JSON.stringify({ error: 'Invalid request' }), {
          status: 400,
          headers: { 'Content-Type': 'application/json' }
        })
      }
    }

    if (urlString.startsWith('/api/run') && (!options || options.method === 'POST')) {
      console.log('Mocking API call to /api/run')
      
      try {
        await new Promise(resolve => setTimeout(resolve, 500))
        
        let requestBody: RunRequestBody = {}
        if (options && options.body) {
          requestBody = JSON.parse(options.body as string)
        }
        
        const isCustomInput = requestBody.custom_input && requestBody.custom_input.trim()
        
        let testCases
        if (isCustomInput) {
          testCases = [
            {
              name: 'Custom Input',
              input: requestBody.custom_input,
              expected_output: 'N/A',
              actual_output: 'Your output here (simulated)'
            }
          ]
        } else {
          testCases = [
            {
              name: 'Sample Test Case 1',
              input: '5\na b c d e',
              expected_output: 'e d c b a',
              actual_output: 'e d c b a'
            },
            {
              name: 'Sample Test Case 2',
              input: '3\nhello world test',
              expected_output: 'test world hello',
              actual_output: 'test world hello'
            }
          ]
        }
        
        const responseData = {
          run_id: Date.now(),
          status: 'Success',
          message: isCustomInput ? 'Custom input executed' : 'Sample test cases passed',
          test_cases: testCases,
          timestamp: Date.now()
        }
        
        return new Response(JSON.stringify(responseData), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        })
      } catch {
        return new Response(JSON.stringify({ error: 'Invalid request' }), {
          status: 400,
          headers: { 'Content-Type': 'application/json' }
        })
      }
    }

    return originalFetch(...args)
  }
}

export {}
