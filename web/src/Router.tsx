import { BrowserRouter, Routes, Route } from 'react-router-dom';
import App from './App';
import { Admin } from './admin/Admin';
import { EventDetailPage } from './pages/EventDetailPage';
import { EntityPage } from './pages/EntityPage';
import { CategoryPage } from './pages/CategoryPage';
import { ApiDocsPage } from './pages/ApiDocsPage';
import { RiskAnalysisPage } from './pages/RiskAnalysisPage';

export function Router() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<App />} />
        <Route path="/forecasts" element={<App />} />
        <Route path="/strategies" element={<App />} />
        <Route path="/admin" element={<Admin />} />
        <Route path="/api-docs" element={<ApiDocsPage />} />
        <Route path="/events/:id" element={<EventDetailPage />} />
        <Route path="/entity/:name" element={<EntityPage />} />
        <Route path="/category/:name" element={<CategoryPage />} />
        <Route path="/market/:ticker" element={<RiskAnalysisPage />} />
      </Routes>
    </BrowserRouter>
  );
}
