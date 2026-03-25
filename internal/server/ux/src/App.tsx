import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom';
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
import FeatureDetailPage from './pages/FeatureDetailPage';
import ExportGeminiPage from './pages/ExportGeminiPage';
import ExportClaudePage from './pages/ExportClaudePage';
import StatisticsPage from './pages/StatisticsPage';
import BacklogPage from './pages/BacklogPage';
import SkillsPage from './pages/SkillsPage';
import DockerfilesPage from './pages/DockerfilesPage';
import AccountPage from './pages/AccountPage';
import ApiKeysPage from './pages/ApiKeysPage';
import NotificationsPage from './pages/NotificationsPage';
import NodesPage from './pages/NodesPage';
import NodeSettingsPage from './pages/NodeSettingsPage';
import FeatureChatPage from './pages/FeatureChatPage';
import SpecializedAgentDetailPage from './pages/SpecializedAgentDetailPage';


function ProtectedRoute() {
  const { isAuthenticated } = useAuth();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <Outlet />;
}

function PublicOnlyRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();
  if (isAuthenticated) return <Navigate to="/" replace />;
  return <>{children}</>;
}

function ProtectedLayout() {
  return (
    <Layout>
      <Outlet />
    </Layout>
  );
}

function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<PublicOnlyRoute><LoginPage /></PublicOnlyRoute>} />
            <Route element={<ProtectedRoute />}>
              <Route element={<ProtectedLayout />}>
                <Route path="/" element={<HomePage />} />
                <Route path="/projects/:projectId" element={<KanbanPage />} />
                <Route path="/projects/:projectId/board" element={<KanbanPage />} />
                <Route path="/projects/:projectId/backlog" element={<BacklogPage />} />
                <Route path="/projects/:projectId/settings" element={<ProjectSettingsPage />} />
                <Route path="/projects/:projectId/settings/agents" element={<ProjectSettingsPage />} />
                <Route path="/projects/:projectId/features" element={<FeaturesPage />} />
                <Route path="/projects/:projectId/features/:featureId" element={<FeatureDetailPage />} />
                <Route path="/projects/:projectId/features/:featureId/chat" element={<FeatureChatPage />} />
                <Route path="/projects/:projectId/export/gemini" element={<ExportGeminiPage />} />
                <Route path="/projects/:projectId/export/claude" element={<ExportClaudePage />} />
                <Route path="/projects/:projectId/statistics" element={<StatisticsPage />} />
                <Route path="/projects/:projectId/roles" element={<RolesPage />} />
                <Route path="/roles" element={<RolesPage />} />
                <Route path="/agents/:parentSlug/specialized/:specSlug" element={<SpecializedAgentDetailPage />} />
                <Route path="/skills" element={<SkillsPage />} />
                <Route path="/dockerfiles" element={<DockerfilesPage />} />
                <Route path="/notifications" element={<NotificationsPage />} />
                <Route path="/nodes" element={<NodesPage />} />
                <Route path="/nodes/:nodeId/settings" element={<NodeSettingsPage />} />
                <Route path="/account" element={<AccountPage />} />
                <Route path="/account/api-keys" element={<ApiKeysPage />} />
              </Route>
            </Route>
          </Routes>
        </AuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;
