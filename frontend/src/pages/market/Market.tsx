import { useNavigate } from 'react-router-dom';
import { useEffect } from 'react';

export default function Market() {
  const navigate = useNavigate();
  useEffect(() => { navigate('/marketplace', { replace: true }); }, [navigate]);
  return null;
}
