/**
 * MSW server setup for Vitest.
 *
 * Lifecycle hooks (beforeAll/afterEach/afterAll) are registered in
 * test-setup.ts which runs as a vitest setupFile, guaranteeing
 * server.listen() is called before every test file.
 *
 * Usage in test files:
 *   import { server } from "./msw-server"
 *   // Override for specific test:
 *   server.use(http.get("http://localhost:8080/api/packages", () =>
 *     HttpResponse.json({ data: [], error: null })
 *   ))
 */

import { setupServer } from "msw/node"
import { handlers } from "./msw-handlers"

export const server = setupServer(...handlers)
