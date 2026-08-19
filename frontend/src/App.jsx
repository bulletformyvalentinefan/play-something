import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import { PlayerProvider } from './context/PlayerContext'
import ProtectedRoute from './components/ProtectedRoute'
import Header from './components/Header'
import PlayerBar from './components/PlayerBar'
import AuthPage from './pages/AuthPage'
import Home from './pages/Home'
import Library from './pages/Library'
import PlaylistDetail from './pages/PlaylistDetail'

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <PlayerProvider>
          <Header />
          <main className="layout">
            <Routes>
              <Route path="/auth" element={<AuthPage />} />
              <Route element={<ProtectedRoute />}>
                <Route path="/" element={<Home />} />
                <Route path="/library" element={<Library />} />
                <Route path="/playlists/:id" element={<PlaylistDetail />} />
              </Route>
              <Route path="*" element={<AuthPage />} />
            </Routes>
          </main>
          <PlayerBar />
        </PlayerProvider>
      </AuthProvider>
    </BrowserRouter>
  )
}