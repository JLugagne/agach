import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { Components } from 'react-markdown';

interface MarkdownContentProps {
  content: string;
  className?: string;
  /** Override text color for links — defaults to var(--primary) */
  linkColor?: string;
}

/**
 * Renders markdown content with GitHub-Flavored Markdown support.
 * Styled to match the dark theme of the kanban UI.
 */
export default function MarkdownContent({ content, className = '', linkColor = 'var(--primary)' }: MarkdownContentProps) {
  const components: Components = {
    h1: ({ children }) => (
      <h1 className="text-[var(--text-primary)] text-lg font-['Newsreader'] font-medium mt-4 mb-2 first:mt-0">{children}</h1>
    ),
    h2: ({ children }) => (
      <h2 className="text-[var(--text-primary)] text-base font-['Newsreader'] font-medium mt-3 mb-1.5 first:mt-0">{children}</h2>
    ),
    h3: ({ children }) => (
      <h3 className="text-[var(--text-secondary)] text-sm font-['Inter'] font-semibold mt-2 mb-1 first:mt-0">{children}</h3>
    ),
    h4: ({ children }) => (
      <h4 className="text-[var(--text-muted)] text-sm font-['Inter'] font-semibold mt-2 mb-1 first:mt-0">{children}</h4>
    ),
    p: ({ children }) => (
      <p className="text-[var(--text-secondary)] text-sm font-['Inter'] leading-relaxed mb-2 last:mb-0">{children}</p>
    ),
    strong: ({ children }) => (
      <strong className="text-[var(--text-primary)] font-semibold">{children}</strong>
    ),
    em: ({ children }) => (
      <em className="text-[var(--text-secondary)] italic">{children}</em>
    ),
    ul: ({ children }) => (
      <ul className="list-disc list-inside space-y-0.5 mb-2 text-[var(--text-secondary)] text-sm font-['Inter'] leading-relaxed pl-2">
        {children}
      </ul>
    ),
    ol: ({ children }) => (
      <ol className="list-decimal list-inside space-y-0.5 mb-2 text-[var(--text-secondary)] text-sm font-['Inter'] leading-relaxed pl-2">
        {children}
      </ol>
    ),
    li: ({ children }) => (
      <li className="text-[var(--text-secondary)] text-sm font-['Inter'] leading-relaxed">{children}</li>
    ),
    // In react-markdown v10, inline code is just <code> without a parent <pre>.
    // Block code is <pre><code>. We style the <pre> wrapper for block code.
    code: ({ children, ...props }) => (
      <code
        className="bg-[var(--bg-elevated)] text-[var(--primary)] text-xs font-['JetBrains_Mono'] px-1 py-0.5 rounded"
        {...props}
      >
        {children}
      </code>
    ),
    pre: ({ children }) => (
      <pre className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded p-3 mb-2 overflow-x-auto [&>code]:bg-transparent [&>code]:p-0 [&>code]:text-xs [&>code]:font-['JetBrains_Mono'] [&>code]:text-[var(--primary)] [&>code]:whitespace-pre">
        {children}
      </pre>
    ),
    blockquote: ({ children }) => (
      <blockquote className="border-l-2 border-[var(--border-primary)] pl-3 my-2 text-[var(--text-muted)] italic text-sm font-['Inter']">
        {children}
      </blockquote>
    ),
    a: ({ children, href }) => (
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className="underline underline-offset-2 hover:opacity-80 transition-opacity"
        style={{ color: linkColor }}
      >
        {children}
      </a>
    ),
    img: ({ src, alt }) => (
      <img
        src={src}
        alt={alt}
        style={{ maxWidth: '100%', borderRadius: '4px', margin: '4px 0', display: 'block' }}
      />
    ),
    hr: () => (
      <hr className="border-[var(--border-primary)] my-3" />
    ),
    table: ({ children }) => (
      <div className="overflow-x-auto mb-2">
        <table className="w-full text-sm font-['Inter'] border-collapse">{children}</table>
      </div>
    ),
    thead: ({ children }) => (
      <thead className="border-b border-[var(--border-primary)]">{children}</thead>
    ),
    th: ({ children }) => (
      <th className="text-left text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider px-2 py-1.5">
        {children}
      </th>
    ),
    td: ({ children }) => (
      <td className="text-[var(--text-secondary)] px-2 py-1.5 border-t border-[var(--border-primary)]">{children}</td>
    ),
    tr: ({ children }) => (
      <tr className="hover:bg-[var(--bg-tertiary)] transition-colors">{children}</tr>
    ),
  };

  // Normalize literal \n sequences (from MCP/JSON double-escaping) to real newlines
  const normalized = content.replace(/\\n/g, '\n');

  return (
    <div className={className}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {normalized}
      </ReactMarkdown>
    </div>
  );
}
