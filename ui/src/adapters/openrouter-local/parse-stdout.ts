import type { StdoutLineParser, TranscriptEntry } from "../types";

export const parseOpenRouterStdoutLine: StdoutLineParser = (
  line: string,
  ts: string,
): TranscriptEntry[] => {
  const trimmed = line.trim();
  if (!trimmed) return [];
  return [{ kind: "text", role: "assistant", content: trimmed, ts }];
};
