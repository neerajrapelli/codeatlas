import { NodeCitation } from './NodeCitation';
import type { RelatedFile } from '../../types';

function renderTextWithCode(text: string, keyPrefix: string) {
  const segments = text.split(/(```[\w-]*\n?[\s\S]*?```)/g);
  return segments.map((segment, i) => {
    const fenced = /^```(\w*-?\w*)?\n?([\s\S]*?)```$/.exec(segment);
    if (fenced) {
      const lang = fenced[1]?.trim();
      const body = fenced[2]?.trimEnd() ?? '';
      return (
        <pre key={`${keyPrefix}-code-${String(i)}`} className="ai-code-block">
          {lang ? <span className="ai-code-block__lang">{lang}</span> : null}
          <code>{body}</code>
        </pre>
      );
    }
    if (!segment) return null;
    const inlineParts = segment.split(/(`[^`\n]+`)/g);
    return (
      <span key={`${keyPrefix}-t-${String(i)}`} className="ai-msg-assistant__text">
        {inlineParts.map((part, j) => {
          const inline = /^`([^`\n]+)`$/.exec(part);
          if (inline) {
            return (
              <code key={`${keyPrefix}-i-${String(i)}-${String(j)}`} className="ai-inline-code">
                {inline[1]}
              </code>
            );
          }
          return part;
        })}
      </span>
    );
  });
}

/** Renders assistant text with optional unverified path markers from guard validation. */
export function AssistantMessage({
  content,
  relatedFiles,
  pathValidation,
  onSelectFile,
}: {
  content: string;
  relatedFiles: RelatedFile[];
  pathValidation?: Record<string, boolean>;
  onSelectFile: (fileId: string, path: string) => void;
}) {
  const parts = content.split(/(⟨unverified:[^⟩]+⟩)/g);
  return (
    <article className="ai-msg-assistant">
      <div className="ai-msg-assistant__body">
        {parts.map((part, i) => {
          const m = /^⟨unverified:(.+?)⟩$/.exec(part);
          if (m) {
            return (
              <span
                key={`u-${String(i)}`}
                className="ai-unverified-mention"
                title="Not found in repository index"
              >
                {m[1]}
              </span>
            );
          }
          return renderTextWithCode(part, `seg-${String(i)}`);
        })}
      </div>
      {relatedFiles.length > 0 ? (
        <div className="ai-citations">
          {relatedFiles.map((f) => (
            <NodeCitation
              key={f.path}
              path={f.path}
              fileId={String(f.fileId)}
              verified={pathValidation?.[f.path] !== false}
              onSelect={onSelectFile}
            />
          ))}
        </div>
      ) : null}
    </article>
  );
}
