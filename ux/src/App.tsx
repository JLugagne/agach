import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { type ReactNode } from 'react';
import { Layout } from './components/Layout';
import { ThemeProvider } from './components/ThemeContext';
import { AuthProvider, useAuth } from './components/AuthContext';
import LoginPage from './pages/LoginPage';
import HomePage from './pages/HomePage';
import KanbanPage from './pages/KanbanPage';
import RolesPage from './pages/RolesPage';
import ProjectSettingsPage from './pages/ProjectSettingsPage';
import FeaturesPage from './pages/FeaturesPage';
import ExportGeminiPage from './pages/ExportGeminiPage';
import ExportClaudePage from './pages/ExportClaudePage';
import StatisticsPage from './pages/StatisticsPage';
import BacklogPage from './pages/BacklogPage';
import SkillsPage from './pages/SkillsPage';
import DockerfilesPage from './pages/DockerfilesPage';
import AccountPage from './pages/AccountPage';
import ApiKeysPage from './pages/ApiKeysPage';

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function PublicOnlyRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();
  if (isAuthenticated) return <Navigate to="/" replace />;
  return <>{children}</>;
}

function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<PublicOnlyRoute><LoginPage /></PublicOnlyRoute>} />
            <Route path="/" element={<ProtectedRoute><Layout><HomePage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId" element={<ProtectedRoute><Layout><KanbanPage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId/board" element={<ProtectedRoute><Layout><KanbanPage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId/backlog" element={<ProtectedRoute><Layout><BacklogPage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId/settings" element={<ProtectedRoute><Layout><ProjectSettingsPage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId/settings/agents" element={<ProtectedRoute><Layout><ProjectSettingsPage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId/features" element={<ProtectedRoute><Layout><FeaturesPage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId/settings/sub-projects" element={<ProtectedRoute><Layout><FeaturesPage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId/export/gemini" element={<ProtectedRoute><Layout><ExportGeminiPage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId/export/claude" element={<ProtectedRoute><Layout><ExportClaudePage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId/statistics" element={<ProtectedRoute><Layout><StatisticsPage /></Layout></ProtectedRoute>} />
            <Route path="/projects/:projectId/roles" element={<ProtectedRoute><Layout><RolesPage /></Layout></ProtectedRoute>} />
            <Route path="/roles" element={<ProtectedRoute><Layout><RolesPage /></Layout></ProtectedRoute>} />
            <Route path="/skills" element={<ProtectedRoute><Layout><SkillsPage /></Layout></ProtectedRoute>} />
            <Route path="/dockerfiles" element={<ProtectedRoute><Layout><DockerfilesPage /></Layout></ProtectedRoute>} />
            <Route path="/account" element={<ProtectedRoute><Layout><AccountPage /></Layout></ProtectedRoute>} />
            <Route path="/account/api-keys" element={<ProtectedRoute><Layout><ApiKeysPage /></Layout></ProtectedRoute>} />
          </Routes>
        </AuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;
