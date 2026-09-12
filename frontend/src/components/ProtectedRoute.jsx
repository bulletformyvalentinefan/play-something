import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function ProtectedRoute() {
  const { user } = useAuth()
  // Solo Spotify cuenta, sin logeo local
  return user?.id ? <Outlet /> : <Navigate to="/auth" replace />
}