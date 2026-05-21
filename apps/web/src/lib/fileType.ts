import type { FileType } from '../types';

export function detectFileType(path: string): FileType {
  const lower = path.toLowerCase();
  if (lower.includes('.test.') || lower.includes('.spec.') || lower.endsWith('_test.ts')) {
    return 'test';
  }
  if (lower.endsWith('.ts') || lower.endsWith('.tsx')) return 'ts';
  if (lower.endsWith('.go')) return 'go';
  if (lower.endsWith('.css') || lower.endsWith('.scss')) return 'css';
  if (
    lower.endsWith('.json') ||
    lower.endsWith('.yaml') ||
    lower.endsWith('.yml') ||
    lower.endsWith('.toml') ||
    lower.endsWith('.config.js')
  ) {
    return 'config';
  }
  return 'other';
}

export function fileTypeLabel(t: FileType): string {
  if (t === 'ts') return 'ts';
  if (t === 'go') return 'go';
  if (t === 'css') return 'css';
  if (t === 'test') return 'test';
  if (t === 'config') return 'cfg';
  return 'file';
}

export function basename(path: string): string {
  const i = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'));
  return i >= 0 ? path.slice(i + 1) : path;
}

export function parentPrefix(path: string): string {
  const norm = path.replace(/\\/g, '/');
  const i = norm.lastIndexOf('/');
  if (i <= 0) return '';
  return norm.slice(0, i);
}
