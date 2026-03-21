import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { ThemeProvider } from './components/ThemeContext';
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

function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Layout><HomePage /></Layout>} />
          <Route path="/projects/:projectId" element={<Layout><KanbanPage /></Layout>} />
          <Route path="/projects/:projectId/board" element={<Layout><KanbanPage /></Layout>} />
          <Route path="/projects/:projectId/backlog" element={<Layout><BacklogPage /></Layout>} />
          <Route path="/projects/:projectId/settings" element={<Layout><ProjectSettingsPage /></Layout>} />
          <Route path="/projects/:projectId/features" element={<Layout><FeaturesPage /></Layout>} />
          <Route path="/projects/:projectId/settings/sub-projects" element={<Layout><FeaturesPage /></Layout>} />
          <Route path="/projects/:projectId/export/gemini" element={<Layout><ExportGeminiPage /></Layout>} />
          <Route path="/projects/:projectId/export/claude" element={<Layout><ExportClaudePage /></Layout>} />
          <Route path="/projects/:projectId/statistics" element={<Layout><StatisticsPage /></Layout>} />
          <Route path="/projects/:projectId/roles" element={<Layout><RolesPage /></Layout>} />
          <Route path="/roles" element={<Layout><RolesPage /></Layout>} />
          <Route path="/skills" element={<Layout><SkillsPage /></Layout>} />
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;
