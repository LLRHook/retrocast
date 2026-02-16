import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { Components } from "react-markdown";
import { useState } from "react";

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(text).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <button
      onClick={handleCopy}
      className="absolute right-2 top-2 rounded bg-white/10 px-2 py-0.5 text-xs text-text-muted hover:bg-white/20 hover:text-text-primary"
    >
      {copied ? "Copied" : "Copy"}
    </button>
  );
}

const components: Components = {
  a({ children, href, ...props }) {
    return (
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className="text-accent hover:underline"
        {...props}
      >
        {children}
      </a>
    );
  },
  code({ children, className, ...props }) {
    const isBlock = className?.startsWith("language-");
    const text = String(children).replace(/\n$/, "");

    if (isBlock) {
      return (
        <div className="group relative my-1">
          <CopyButton text={text} />
          <pre className="overflow-x-auto rounded bg-bg-secondary p-3 text-sm">
            <code className={`font-mono ${className}`} {...props}>
              {children}
            </code>
          </pre>
        </div>
      );
    }

    return (
      <code
        className="rounded bg-black/30 px-1 py-0.5 text-sm font-mono text-text-primary"
        {...props}
      >
        {children}
      </code>
    );
  },
  pre({ children }) {
    // pre is handled by the code component for blocks
    return <>{children}</>;
  },
  blockquote({ children }) {
    return (
      <blockquote className="my-1 border-l-4 border-text-muted pl-3 text-text-secondary">
        {children}
      </blockquote>
    );
  },
  ul({ children }) {
    return <ul className="ml-4 list-disc">{children}</ul>;
  },
  ol({ children }) {
    return <ol className="ml-4 list-decimal">{children}</ol>;
  },
  p({ children }) {
    return <p className="my-0">{children}</p>;
  },
};

export default function MarkdownContent({ content }: { content: string }) {
  return (
    <div className="text-sm text-text-secondary [&_del]:text-text-muted [&_em]:italic [&_strong]:font-semibold [&_strong]:text-text-primary">
      <Markdown remarkPlugins={[remarkGfm]} components={components}>
        {content}
      </Markdown>
    </div>
  );
}
