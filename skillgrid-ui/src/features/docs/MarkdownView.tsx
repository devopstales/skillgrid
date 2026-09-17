import { useMemo } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize'
import { MermaidBlock } from './MermaidBlock'

// GFM needs task-list checkboxes + tables to survive sanitize; extend the
// default schema with the handful of tags/attrs a doc might use.
const sanitizeSchema = {
  ...defaultSchema,
  attributes: {
    ...defaultSchema.attributes,
    '*': ['id', 'align'],
    input: ['type', 'checked', 'disabled'],
    img: ['src', 'alt', 'title'],
  },
}

// slugify mirrors GitHub-style anchor ids so the TOC scroll-spy can target them.
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/\s+/g, '-')
}

interface Props {
  body: string
  securityLevel?: string
}

export function MarkdownView({ body, securityLevel }: Props) {
  const components = useMemo(
    () => ({
      // Capture ```mermaid fences and render them as MermaidBlock.
      code({ className, children }: { className?: string; children?: React.ReactNode }) {
        const text = String(children ?? '').replace(/\n$/, '')
        if (className?.includes('language-mermaid')) {
          return <MermaidBlock code={text} securityLevel={securityLevel} />
        }
        return (
          <code className={className}>
            {children}
          </code>
        )
      },
      // Pre-wrap the fenced block container so mermaid SVG has room.
      pre({ children }: { children?: React.ReactNode }) {
        return <pre className="overflow-x-auto rounded-md border border-edge bg-black/30 p-3 text-sm">{children}</pre>
      },
      h1: ({ children }: { children?: React.ReactNode }) => (
        <h1 id={slugify(String(children ?? ''))} className="mb-3 mt-6 border-b border-edge pb-2 text-2xl font-semibold text-zinc-100 first:mt-0">{children}</h1>
      ),
      h2: ({ children }: { children?: React.ReactNode }) => (
        <h2 id={slugify(String(children ?? ''))} className="mb-2 mt-6 text-xl font-semibold text-zinc-100">{children}</h2>
      ),
      h3: ({ children }: { children?: React.ReactNode }) => (
        <h3 id={slugify(String(children ?? ''))} className="mb-2 mt-4 text-lg font-semibold text-zinc-100">{children}</h3>
      ),
      a: ({ href, children }: { href?: string; children?: React.ReactNode }) => (
        <a href={href} target="_blank" rel="noreferrer" className="text-accent underline decoration-accent/40 hover:decoration-accent">
          {children}
        </a>
      ),
      table: ({ children }: { children?: React.ReactNode }) => (
        <table className="my-3 w-full border-collapse text-sm">{children}</table>
      ),
      th: ({ children }: { children?: React.ReactNode }) => (
        <th className="border border-edge px-2 py-1 text-left font-medium text-zinc-200">{children}</th>
      ),
      td: ({ children }: { children?: React.ReactNode }) => (
        <td className="border border-edge px-2 py-1 text-zinc-300">{children}</td>
      ),
      blockquote: ({ children }: { children?: React.ReactNode }) => (
        <blockquote className="my-3 border-l-2 border-accent/50 pl-3 text-zinc-400">{children}</blockquote>
      ),
      ul: ({ children }: { children?: React.ReactNode }) => (
        <ul className="my-2 list-disc space-y-1 pl-5 text-zinc-300">{children}</ul>
      ),
      ol: ({ children }: { children?: React.ReactNode }) => (
        <ol className="my-2 list-decimal space-y-1 pl-5 text-zinc-300">{children}</ol>
      ),
    }),
    [securityLevel],
  )

  return (
    <div className="max-w-3xl text-sm leading-relaxed">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[[rehypeSanitize, sanitizeSchema]]}
        components={components}
      >
        {body}
      </ReactMarkdown>
    </div>
  )
}
