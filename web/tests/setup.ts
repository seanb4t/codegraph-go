// 03-01 (BRW-06): global vitest setup, loaded via vite.config.ts's
// test.setupFiles. Registers jsdom-aware DOM matchers
// (toBeInTheDocument, etc.) and @testing-library/svelte's own vitest
// integration (auto-cleanup after each test) once, for every test file
// under web/tests/.
import '@testing-library/jest-dom/vitest';
import '@testing-library/svelte/vitest';
