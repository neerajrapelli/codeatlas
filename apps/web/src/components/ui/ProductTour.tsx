import { useStore } from '../../store';

const STEPS = [
  {
    title: 'Add a repository',
    body: 'Paste a GitHub URL in Repositories to start indexing. Progress appears in the status bar.',
    target: 'repos' as const,
  },
  {
    title: 'Explore the architecture map',
    body: 'Drill into clusters on the canvas. Right-click a file for blast radius analysis.',
    target: 'map' as const,
  },
  {
    title: 'Ask the architecture assistant',
    body: 'Use the AI panel for grounded questions about impact, ownership, and dependencies.',
    target: null,
  },
];

export function ProductTour() {
  const step = useStore((s) => s.tourStep);
  const setTourStep = useStore((s) => s.setTourStep);
  const completeTour = useStore((s) => s.completeTour);
  const setSidebarView = useStore((s) => s.setSidebarView);
  const toggleBottomPanel = useStore((s) => s.toggleBottomPanel);
  const bottomPanelOpen = useStore((s) => s.bottomPanelOpen);

  if (step == null || step < 0 || step >= STEPS.length) return null;

  const current = STEPS[step]!;

  const next = () => {
    if (current.target) setSidebarView(current.target);
    if (step === 2 && !bottomPanelOpen) toggleBottomPanel();
    if (step >= STEPS.length - 1) {
      completeTour();
      return;
    }
    setTourStep(step + 1);
  };

  const skip = () => completeTour();

  return (
    <div className="product-tour-overlay" role="dialog" aria-modal="true" aria-labelledby="tour-title">
      <div className="product-tour-card">
        <p className="product-tour-card__step">
          Step {step + 1} of {STEPS.length}
        </p>
        <h2 id="tour-title" className="product-tour-card__title">
          {current.title}
        </h2>
        <p className="product-tour-card__body">{current.body}</p>
        <div className="product-tour-card__actions">
          <button type="button" className="btn-secondary" onClick={skip}>
            Skip tour
          </button>
          <button type="button" className="btn-primary" onClick={next}>
            {step >= STEPS.length - 1 ? 'Get started' : 'Next'}
          </button>
        </div>
      </div>
    </div>
  );
}
