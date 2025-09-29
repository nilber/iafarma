import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import { cn } from '@/lib/utils';
import { MediaMessage } from './media-message';
import 'highlight.js/styles/github.css';

interface MessageContentProps {
  content: string;
  type?: string;
  mediaUrl?: string;
  mediaType?: string;
  filename?: string;
  className?: string;
}

export function MessageContent({ 
  content, 
  type, 
  mediaUrl, 
  mediaType, 
  filename, 
  className 
}: MessageContentProps) {
  // Render media message if it's not a text message and has media
  if (type && type !== 'text' && mediaUrl) {
    return (
      <MediaMessage
        type={type}
        mediaUrl={mediaUrl}
        mediaType={mediaType}
        filename={filename}
        content={content}
        className={className}
      />
    );
  }

  // Fallback to text rendering
  return (
    <div className={cn("prose prose-sm max-w-none", className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        components={{
          // Customizar componentes específicos do Markdown
          p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
          ul: ({ children }) => <ul className="list-disc ml-4 mb-2">{children}</ul>,
          ol: ({ children }) => <ol className="list-decimal ml-4 mb-2">{children}</ol>,
          li: ({ children }) => <li className="mb-1">{children}</li>,
          strong: ({ children }) => <strong className="font-semibold text-gray-900">{children}</strong>,
          em: ({ children }) => <em className="italic">{children}</em>,
          code: ({ children, className }) => {
            const isInline = !className;
            return isInline ? (
              <code className="px-1 py-0.5 bg-gray-100 rounded text-sm font-mono">
                {children}
              </code>
            ) : (
              <code className={className}>{children}</code>
            );
          },
          pre: ({ children }) => (
            <pre className="bg-gray-50 border rounded-md p-3 text-sm overflow-x-auto mb-2">
              {children}
            </pre>
          ),
          blockquote: ({ children }) => (
            <blockquote className="border-l-4 border-blue-300 pl-4 py-2 bg-blue-50 mb-2">
              {children}
            </blockquote>
          ),
          h1: ({ children }) => <h1 className="text-lg font-bold mb-2">{children}</h1>,
          h2: ({ children }) => <h2 className="text-base font-bold mb-2">{children}</h2>,
          h3: ({ children }) => <h3 className="text-sm font-bold mb-1">{children}</h3>,
          // Tornar links clicáveis
          a: ({ href, children }) => (
            <a 
              href={href} 
              target="_blank" 
              rel="noopener noreferrer"
              className="text-blue-600 hover:text-blue-800 underline"
            >
              {children}
            </a>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
