import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import Harness from './fixtures/Harness.svelte';

describe('harness GREEN proof', () => {
	it('renders a real compiled .svelte fixture under jsdom, with DOM matchers registered', () => {
		render(Harness, { props: { label: 'codegraph-harness-proof' } });

		expect(screen.getByTestId('harness-label')).toBeInTheDocument();
		expect(screen.getByTestId('harness-label')).toHaveTextContent('codegraph-harness-proof');
	});
});
