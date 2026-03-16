import { Link, useParams } from 'react-router-dom';
import { ChevronLeft, Sparkles } from 'lucide-react';

export default function ExportGeminiPage() {
  const { projectId } = useParams<{ projectId: string }>();

  return (
    <div className="min-h-screen bg-[#0F0F0F] flex flex-col items-center justify-center px-8">
      <div className="text-center max-w-md">
        <div className="mx-auto w-16 h-16 rounded-full bg-[#1A1A1A] flex items-center justify-center mb-6">
          <Sparkles size={28} className="text-[#FFD060]" />
        </div>
        <h1 className="font-heading text-2xl text-[#F0F0F0] mb-3">Export to Gemini</h1>
        <p className="text-sm text-[#888888] mb-8">
          This feature is coming soon. It will allow you to export your project context and tasks
          for use with Google Gemini.
        </p>
        <Link
          to={projectId ? `/projects/${projectId}` : '/'}
          className="inline-flex items-center gap-1.5 text-sm text-[#00C896] hover:text-[#00C896]/80 transition-colors"
        >
          <ChevronLeft size={14} />
          Back to Project
        </Link>
      </div>
    </div>
  );
}
