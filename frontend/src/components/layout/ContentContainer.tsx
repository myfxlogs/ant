import { useLocation } from 'react-router-dom';

const FLUID_ROUTES = ['/strategy/workspace', '/strategy/new'];

export default function ContentContainer({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const isFluid = FLUID_ROUTES.some(route => location.pathname.startsWith(route)) ||
    /^\/strategy\/[^/]+\/edit$/.test(location.pathname);

  return (
    <div
      style={{
        maxWidth: isFluid ? undefined : 1360,
        margin: '0 auto',
        width: '100%',
        padding: isFluid ? '0 12px' : undefined,
      }}
    >
      {children}
    </div>
  );
}
